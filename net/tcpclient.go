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
	"nbpy/codecs"
	"nbpy/packets"
)

type TCPClient struct {
	DataController
	Codec          *codecs.Codec
	Format         *packets.PacketFormat
	dataNotifyChan chan int
	controller     *TCPController
}

func CreateTCPClient(format *packets.PacketFormat, codec *codecs.Codec) (*TCPClient) {
	srv := new(TCPClient)
	srv.Codec = codec
	srv.Format = format
	return srv
}

func (receiver *TCPClient) Connect(addr string, port int) (error) {
	address := fmt.Sprintf("%s:%d", addr, port)
	if port == 0 {
		address = addr
	}
	_, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}
	conn, err := net.Dial("tcp", address)
	if err != nil {
		utils.LogError("### 连接 %s 失败. err: %s", address, err)
		return err
	}

	receiver.processClient(conn)

	utils.LogInfo("### 连接 %s 成功", address)

	return err
}

func (receiver *TCPClient) processClient(conn net.Conn) {

	dataRW := createDataReadWriter(receiver.Codec, receiver.Format)
	dataRW.OnDataDecoded = receiver.OnDataDecoded
	receiver.controller = createTCPController(conn, dataRW)
	receiver.controller.Schedule()

}

func (receiver *TCPClient) Close() {
	if receiver.controller != nil {
		receiver.controller.Close()
		receiver.controller = nil
	}
}

func (receiver *TCPClient) Send(data ...codecs.IMData) {
	if receiver.controller != nil {
		receiver.controller.Send(data...)
	}
}