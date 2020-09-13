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

package net

import (
    "github.com/packing/nbpy/utils"
    "sync"
    "net"
    "github.com/packing/nbpy/codecs"
    "github.com/packing/nbpy/errors"
    "syscall"
    "runtime"
    "encoding/binary"
)

/*
goroutine 1 => process read
goroutine 2 => process data
*/

type UnixMsgData struct {
    addr string
    b []byte
    oob []byte
}

type UnixMsgController struct {
    OnStop        OnControllerStop
    id            SessionID
    ioinner       net.UnixConn

    queue         chan UnixMsgData
    associatedObject interface{}
}

func createUnixMsgController(ioSrc net.UnixConn) (*UnixMsgController) {
    sor := new(UnixMsgController)
    sor.ioinner = ioSrc
    sor.id = NewSessionID()
    sor.associatedObject = nil
    return sor
}

func (receiver *UnixMsgController) SetAssociatedObject(o interface{}) {
    receiver.associatedObject = o
}

func (receiver *UnixMsgController) GetAssociatedObject() (interface{}) {
    return receiver.associatedObject
}

func (receiver UnixMsgController) GetSource() (string){
    return receiver.ioinner.LocalAddr().String()
}

func (receiver UnixMsgController) GetSessionID() (SessionID){
    return receiver.id
}

func (receiver UnixMsgController) Close() {
    receiver.ioinner.Close()
}

func (receiver UnixMsgController) Discard() {
}

func (receiver UnixMsgController) CloseOnSended() {
}

func (receiver UnixMsgController) Read(l int) ([]byte, int) {
    return nil, 0
}

func (receiver UnixMsgController) Peek(l int) ([]byte, int) {
    return nil, 0
}

func (receiver UnixMsgController) Write(data []byte) {
}

func (receiver UnixMsgController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) {
    return msg, errors.ErrorDataSentIncomplete
}

func (receiver UnixMsgController) ReadFrom() (string, []byte, int) {
    return "", nil, 0
}

func (receiver *UnixMsgController) WriteTo(addr string, data []byte) {
}

func (receiver *UnixMsgController) clearSendBuffer(addr string) {
}

func (receiver *UnixMsgController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) {
    return nil, nil
}

func (receiver UnixMsgController) SendFdTo(addr string, fds...int) (error) {
    unixAddr, err := net.ResolveUnixAddr("unixgram", addr)
    if err != nil {
        return err
    }

    b := make([]byte, 4)
    binary.BigEndian.PutUint32(b, uint32(len(fds)))

    for _, fd := range fds {
        oob := syscall.UnixRights(fd)
        _, _, err = receiver.ioinner.WriteMsgUnix(b, oob, unixAddr)
        if err != nil {
            return err
        }
    }
    return err
}

func (receiver *UnixMsgController) processData(group *sync.WaitGroup) {
    defer utils.LogPanic(recover())
    for {
        msg, ok := <- receiver.queue
        if !ok {
            break
        }
        fdctrl, ok := receiver.associatedObject.(FileHandleController)
        if !ok {
            continue
        }

        scms, err := syscall.ParseSocketControlMessage(msg.oob)
        if err != nil {
            continue
        }
        for _, cms := range scms {
            fds, err := syscall.ParseUnixRights(&cms)
            if err != nil {
                continue
            }

            for _, fd := range fds {
                go func() {
                    fdctrl.OnFileHandleReceived(fd)
                }()
            }

            IncTotalHandleRecvSize(len(fds))
        }
        runtime.Gosched()
    }

    group.Done()
}

func (receiver *UnixMsgController) processRead(group *sync.WaitGroup) {
    defer utils.LogPanic(recover())
    for {
        b := make([]byte, 4)
        oob := make([]byte, 1024)
        bn, oobn, _, addr, err := receiver.ioinner.ReadMsgUnix(b, oob)
        if err != nil {
            break
        }
        msg := UnixMsgData{addr: addr.String(), b: b[:bn], oob: oob[:oobn]}
        receiver.queue <- msg
        runtime.Gosched()
    }

    close(receiver.queue)
    group.Done()
}

func (receiver *UnixMsgController) Schedule() {
    receiver.queue = make(chan UnixMsgData, 10240)
    group := new(sync.WaitGroup)
    group.Add(2)

    go func() {
        go receiver.processData(group)
        go receiver.processRead(group)
        group.Wait()
        if receiver.OnStop != nil {
            receiver.OnStop(receiver)
        }
        utils.LogError(">>> UNIX消息控制器 %s 已关闭调度", receiver.GetSource())
    }()
}