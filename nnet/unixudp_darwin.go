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
    "github.com/packing/nbpy/codecs"
    "github.com/packing/nbpy/errors"
    "github.com/packing/nbpy/packets"
    "github.com/packing/nbpy/utils"
    "net"
    "os"
)

type UnixUDP struct {
    DataController
    Codec          *codecs.Codec
    Format         *packets.PacketFormat
    dataNotifyChan chan int
    controller     *UnixController
    isClosed       bool
    addr           string

    associatedObject interface{}
}

func CreateUnixUDPWithFormat(format *packets.PacketFormat, codec *codecs.Codec) *UnixUDP {
    s := new(UnixUDP)
    s.Codec = codec
    s.Format = format
    s.isClosed = true
    return s
}

func (receiver *UnixUDP) SetControllerAssociatedObject(o interface{}) {
    receiver.associatedObject = o
}

func (receiver *UnixUDP) Bind(addr string) error {
    receiver.isClosed = true
    unixAddr, err := net.ResolveUnixAddr("unixgram", addr)
    if err != nil {
        return err
    }

    unixConn, err := net.ListenUnixgram("unixgram", unixAddr)
    if err != nil {
        return err
    }

    receiver.addr = addr
    receiver.isClosed = false
    receiver.processClient(*unixConn)

    return nil
}

func (receiver UnixUDP) GetBindAddr() string {
    return receiver.addr
}

func (receiver *UnixUDP) processClient(conn net.UnixConn) {

    dataRW := createDataReadWriter(receiver.Codec, receiver.Format)
    dataRW.OnDataDecoded = receiver.OnDataDecoded
    receiver.controller = createUnixController(conn, dataRW)
    receiver.controller.SetAssociatedObject(receiver.associatedObject)

    receiver.controller.OnStop = func(controller Controller) error {
        utils.LogInfo("unix端口 %s 已经退出监听", controller.GetSessionID())
        receiver.controller = nil
        receiver.isClosed = true
        return nil
    }

    receiver.controller.Schedule()

}

func (receiver *UnixUDP) SendTo(addr string, msgs ...codecs.IMData) ([]codecs.IMData, error) {
    if receiver.isClosed {
        return msgs, errors.ErrorDataSentIncomplete
    }
    _, err := os.Stat(addr)
    if err == nil || !os.IsNotExist(err) {
        return receiver.controller.SendTo(addr, msgs...)
    } else {
        utils.LogError("无法向 %s 发送数据, 请确认它是否仍存在", addr)
        return msgs, err
    }
}

func (receiver *UnixUDP) SendFileHandler(addr string, fds ...int) error {
    return receiver.controller.SendFdTo(addr, fds...)
}

func (receiver *UnixUDP) Close() {
    if !receiver.isClosed {
        //receiver.controller.Close()
        receiver.controller.CloseOnSended()
        receiver.isClosed = true
    }
}
