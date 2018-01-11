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
	"nbpy/utils"
	"nbpy/codecs"
	"nbpy/packets"
	"nbpy/errors"
	"net"
	"fmt"
)

type UDP struct {
	DataController
	Codec          *codecs.Codec
	Format         *packets.PacketFormat
	dataNotifyChan chan int
	controller     *UDPController
}

func CreateUDP(format *packets.PacketFormat, codec *codecs.Codec) (*UDP) {
	s := new(UDP)
	s.Codec = codec
	s.Format = format
	return s
}

func (receiver *UDP) Bind(addr string, port int) (error) {
	address := fmt.Sprintf("%s:%d", addr, port)
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	receiver.processClient(*udpConn)

	return nil
}

func (receiver *UDP) processClient(conn net.UDPConn) {

	dataRW := createDataReadWriter(receiver.Codec, receiver.Format)
	dataRW.OnDataDecoded = receiver.OnDataDecoded
	receiver.controller = createUDPController(conn, dataRW)
	receiver.controller.OnStop = func(controller Controller) error {
		utils.LogInfo("udp端口 %s 已经退出监听", controller.GetSessionID())
		receiver.controller = nil
		return nil
	}
	receiver.controller.Schedule()

}

func (receiver *UDP) SendTo(addr string, port int, msgs...codecs.IMData) ([]codecs.IMData, error) {
	if receiver.controller == nil {
		return msgs, errors.ErrorDataSentIncomplete
	}
	address := fmt.Sprintf("%s:%d", addr, port)
	return receiver.controller.SendTo(address, msgs...)
}

func (receiver *UDP) Close() {
	if receiver.controller != nil {
		receiver.controller.Close()
		receiver.controller = nil
	}
}