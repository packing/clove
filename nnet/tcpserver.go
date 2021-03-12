/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nnet

import (
    "fmt"
    "github.com/packing/nbpy/codecs"
    "github.com/packing/nbpy/errors"
    "github.com/packing/nbpy/packets"
    "github.com/packing/nbpy/utils"
    "net"
    "os"
    "strings"
    "sync"
    "sync/atomic"
)

/*
goroutine 1 => process accept
goroutine 2 => process data unpack/decode/logic-make/encode/pack
*/

type TCPSend struct {
    sessionId SessionID
    msgs      []codecs.IMData
}

type TCPServer struct {
    DataController
    SocketController
    Codec             *codecs.Codec
    Format            *packets.PacketFormat
    limit             int64
    total             int64
    listener          *net.TCPListener
    controllers       *sync.Map
    isClosed          bool
    handleTransfer    *UnixMsg
    handleReceiveAddr string
    OnConnectAccepted func(conn net.Conn) error
    ControllerCome    OnControllerCome
    sendChan          chan *TCPSend
    mutex             sync.Mutex
}

func CreateTCPServer() *TCPServer {
    srv := new(TCPServer)
    srv.handleTransfer = nil
    return srv
}

func CreateTCPServerWithLimit(limit int64) *TCPServer {
    srv := CreateTCPServer()
    srv.limit = limit
    srv.isClosed = true
    return srv
}

func (receiver *TCPServer) SetHandleTransfer(dest string, transfer *UnixMsg) {
    receiver.handleTransfer = transfer
    receiver.handleReceiveAddr = dest
}

func (receiver *TCPServer) GetTotal() int {
    var i = 0
    receiver.controllers.Range(func(key, value interface{}) bool {
        i += 1
        return true
    })
    return i
}

func (receiver TCPServer) OnFileHandleReceived(fd int) error {
    return receiver.processClientFromFileHandle(fd)
}

func (receiver *TCPServer) Bind(addr string, port int) error {
    receiver.isClosed = true
    address := fmt.Sprintf("%s:%d", addr, port)
    if port == 0 {
        address = addr
    }
    tcpAddr, err := net.ResolveTCPAddr("tcp", address)
    if err != nil {
        return err
    }
    receiver.listener, err = net.ListenTCP("tcp", tcpAddr)
    if err != nil {
        utils.LogError("### 监听 %s 失败. err: %s", address, err)
        return err
    }

    receiver.isClosed = false
    receiver.controllers = new(sync.Map)

    utils.LogInfo("### 监听 %s 成功", address)

    return err
}

func (receiver *TCPServer) ServeWithoutListener() error {
    receiver.isClosed = false
    receiver.controllers = new(sync.Map)

    //只有实际服务器才有下发需求，才需要初始化发送队列
    receiver.sendChan = make(chan *TCPSend, 128)

    go receiver.goroutineSend()
    utils.LogInfo("### 无监听服务启动成功")

    return nil
}

func (receiver *TCPServer) addController(controller *TCPController) {
    receiver.controllers.Store(controller.GetSessionID(), controller)
}

func (receiver *TCPServer) delController(controller Controller) {
    receiver.controllers.Delete(controller.GetSessionID())
}

func (receiver *TCPServer) getController(sessid SessionID) *TCPController {
    v, ok := receiver.controllers.Load(sessid)
    if !ok {
        return nil
    }
    conn, ok := v.(*TCPController)
    if !ok {
        return nil
    }
    return conn
}

func (receiver *TCPServer) eachControllers(fn func(*TCPController)) {
    receiver.controllers.Range(func(key, value interface{}) bool {
        conn, ok := value.(*TCPController)
        if !ok {
            return false
        }
        fn(conn)
        return true
    })
}

func (receiver TCPServer) GetController(sessid SessionID) *TCPController {
    return receiver.getController(sessid)
}

func (receiver *TCPServer) processClientFromFileHandle(fd int) error {

    //utils.LogInfo(">>> 接收到转移来到新连接句柄 %d", fd)

    f := os.NewFile(uintptr(fd), "fd-from-old")
    fc, err := net.FileConn(f)
    if err != nil {
        utils.LogError("构造连接对象失败", err)
        return err
    }

    atomic.AddInt64(&receiver.total, 1)

    dataRW := createDataReadWriter(receiver.Codec, receiver.Format)
    dataRW.OnDataDecoded = receiver.OnDataDecoded
    controller := createTCPController(fc, dataRW)

    controller.OnStop = func(controller Controller) error {
        if receiver.OnBye != nil {
            receiver.OnBye(controller)
        }
        atomic.AddInt64(&receiver.total, -1)
        receiver.delController(controller)
        return nil
    }

    if receiver.ControllerCome != nil {
        if err = receiver.ControllerCome(controller); err != nil {
            atomic.AddInt64(&receiver.total, -1)
            return err
        }
    }

    receiver.addController(controller)

    controller.Schedule()

    if receiver.OnWelcome != nil {
        receiver.OnWelcome(controller)
    }

    return nil
}

func (receiver *TCPServer) processClient(conn net.Conn) {
    if receiver.limit > 0 && receiver.total >= receiver.limit {
        conn.Close()
        return
    }
    atomic.AddInt64(&receiver.total, 1)

    dataRW := createDataReadWriter(receiver.Codec, receiver.Format)
    dataRW.OnDataDecoded = receiver.OnDataDecoded
    controller := createTCPController(conn, dataRW)

    controller.OnStop = func(controller Controller) error {
        if receiver.OnBye != nil {
            receiver.OnBye(controller)
        }
        atomic.AddInt64(&receiver.total, -1)
        receiver.delController(controller)
        return nil
    }

    if receiver.ControllerCome != nil {
        if err := receiver.ControllerCome(controller); err != nil {
            atomic.AddInt64(&receiver.total, -1)
            return
        }
    }

    receiver.addController(controller)

    controller.Schedule()

    if receiver.OnWelcome != nil {
        receiver.OnWelcome(controller)
    }
}

func (receiver *TCPServer) goroutineAccept() {
    defer utils.LogPanic(recover())

    if receiver.OnConnectAccepted == nil {
        //只有实际服务器才有下发需求，才需要初始化发送队列
        receiver.sendChan = make(chan *TCPSend, 128)
    }

    for {
        conn, err := receiver.listener.Accept()
        if err != nil {
            if strings.Contains(err.Error(), "use of closed network connection") {
                utils.LogWarn("=== 监听端口已经关闭")
                break
            } else {
                utils.LogError("=== 接收新连接请求失败. 原因: %s", err)
            }
            continue
        }

        go func() {
            if receiver.OnConnectAccepted == nil {
                receiver.processClient(conn)
            } else {
                receiver.OnConnectAccepted(conn)
                conn.Close()
            }
        }()
    }
    utils.LogError("=== 新连接接收器已关闭")
}

func (receiver *TCPServer) Close() {
    if !receiver.isClosed {
        receiver.listener.Close()
        receiver.closeAllController()
        receiver.isClosed = true
    }
}

func (receiver *TCPServer) goroutineSend() {
    defer utils.LogPanic(recover())

    for !receiver.isClosed {
        ts, ok := <-receiver.sendChan
        if ok {
            if ts.sessionId > 0 {
                ctrl := receiver.getController(ts.sessionId)
                if ctrl != nil {
                    ctrl.RawSend(ts.msgs...)
                }
            } else {
                receiver.eachControllers(func(controller *TCPController) {
                    controller.RawSend(ts.msgs...)
                })
            }
        } else {
            //下发队列已销毁,退出发送处理
            break
        }
    }
    close(receiver.sendChan)
}

func (receiver *TCPServer) Schedule() {
    go receiver.goroutineAccept()
    go receiver.goroutineSend()
}

func (receiver *TCPServer) closeAllController(msg ...codecs.IMData) {
    if receiver.isClosed {
        return
    }
    receiver.eachControllers(func(controller *TCPController) {
        controller.Close()
    })
}

func (receiver *TCPServer) CloseController(sessionid SessionID) error {
    processor := receiver.getController(sessionid)
    if processor == nil {
        return errors.ErrorSessionIsNotExists
    }
    processor.Close()
    return nil
}

func (receiver *TCPServer) Send(sessionid SessionID, msg ...codecs.IMData) ([]codecs.IMData, error) {
    if receiver.isClosed {
        return msg, errors.ErrorSessionIsNotExists
    }
    ts := TCPSend{sessionId: sessionid, msgs: msg}
    go func() {
        receiver.sendChan <- &ts
    }()

    return []codecs.IMData{}, nil
}

func (receiver *TCPServer) Mutilcast(sessionids []SessionID, msg ...codecs.IMData) {
    if receiver.isClosed {
        return
    }

    for _, sessionid := range sessionids {
        controller := receiver.getController(sessionid)
        if controller == nil {
            continue
        }
        receiver.Send(sessionid, msg...)
    }
}

func (receiver *TCPServer) Boardcast(msg ...codecs.IMData) {
    if receiver.isClosed {
        return
    }

    receiver.Send(0, msg...)
}
