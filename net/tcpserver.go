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
	"net"
	"fmt"
	"nbpy/utils"
	"strings"
	"nbpy/codecs"
	"nbpy/packets"
	"nbpy/errors"
	"sync"
)

/*
goroutine 1 => process accept
goroutine 2 => process data unpack/decode/logic-make/encode/pack
*/


type TCPServer struct {
	DataController
	SocketController
	Codec          *codecs.Codec
	Format         *packets.PacketFormat
	limit          int
	total          int
	listener       *net.TCPListener
	dataNotifyChan chan int
	controllers    map[SessionID] *TCPController
	mutex          sync.Mutex
}

func CreateTCPServer() (*TCPServer) {
	srv := new(TCPServer)
	return srv
}

func CreateTCPServerWithLimit(limit int) (*TCPServer) {
	srv := CreateTCPServer()
	srv.limit = limit
	return srv
}

func (receiver *TCPServer) Bind(addr string, port int) (error) {
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

	//初始化所有channel
	receiver.dataNotifyChan = make(chan int)
	receiver.controllers = make(map[SessionID] *TCPController)

	utils.LogInfo("### 监听 %s 成功", address)

	return err
}

func (receiver *TCPServer) addController(controller *TCPController) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	receiver.controllers[controller.GetSessionID()] = controller
}

func (receiver *TCPServer) delController(controller Controller) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	delete(receiver.controllers, controller.GetSessionID())
}

func (receiver *TCPServer) processClient(conn net.Conn) {
	if receiver.limit > 0 && receiver.total >= receiver.limit {
		conn.Close()
		return
	}
	receiver.total += 1

	dataRW := createDataReadWriter(receiver.Codec, receiver.Format)
	dataRW.OnDataDecoded = receiver.OnDataDecoded
	controller := createTCPController(conn, dataRW)

	controller.OnStop = func(controller Controller) error {
		if receiver.OnBye != nil {
			receiver.OnBye(controller)
		}
		receiver.total -= 1
		receiver.delController(controller)
		return nil
	}
	go func() {
		<- receiver.dataNotifyChan
		controller.Close()
	}()

	receiver.addController(controller)

	controller.Schedule()

	if receiver.OnWelcome != nil {
		receiver.OnWelcome(controller)
	}
}

func (receiver *TCPServer) goroutineAccept() {
	defer utils.LogPanic()
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

		receiver.processClient(conn)
	}
}

func (receiver *TCPServer) Close() {
	receiver.listener.Close()
	close(receiver.dataNotifyChan)
}

func (receiver *TCPServer) Schedule() {
	go receiver.goroutineAccept()
}

func (receiver *TCPServer) CloseController(sessionid SessionID) (error) {
	processor, ok := receiver.controllers[sessionid]
	if !ok {
		return errors.ErrorSessionIsNotExists
	}
	processor.Close()
	return nil
}


func (receiver *TCPServer) Send(sessionid SessionID,msg ...codecs.IMData) ([]codecs.IMData, error) {
	processor, ok := receiver.controllers[sessionid]
	if !ok {
		return msg, errors.ErrorSessionIsNotExists
	}
	return processor.Send(msg...)
}

func (receiver *TCPServer) Mutilcast(sessionids []SessionID,msg ...codecs.IMData) {
	for _, sessionid := range sessionids {
		processor, ok := receiver.controllers[sessionid]
		if !ok {
			continue
		}
		processor.Send(msg...)
	}
}

func (receiver *TCPServer) Boardcast(msg ...codecs.IMData) {
	for _, controller := range receiver.controllers {
		controller.Send(msg...)
	}
}