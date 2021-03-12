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
    "github.com/packing/nbpy/errors"
    "github.com/packing/nbpy/utils"
    "net"
)

type UnixMsg struct {
    controller *UnixMsgController
    isClosed   bool

    associatedObject interface{}
}

func CreateUnixMsg() *UnixMsg {
    s := new(UnixMsg)
    s.isClosed = true
    return s
}

func (receiver *UnixMsg) SetControllerAssociatedObject(o interface{}) {
    receiver.associatedObject = o
}

func (receiver *UnixMsg) Bind(addr string) error {
    receiver.isClosed = true
    unixAddr, err := net.ResolveUnixAddr("unixgram", addr)
    if err != nil {
        return err
    }

    unixConn, err := net.ListenUnixgram("unixgram", unixAddr)
    if err != nil {
        return err
    }

    receiver.isClosed = false
    receiver.processClient(*unixConn)

    return nil
}

func (receiver *UnixMsg) processClient(conn net.UnixConn) {
    receiver.controller = createUnixMsgController(conn)
    receiver.controller.SetAssociatedObject(receiver.associatedObject)

    receiver.controller.OnStop = func(controller Controller) error {
        utils.LogInfo(">>> unix消息端口 %d 已经退出监听", controller.GetSessionID())
        receiver.controller = nil
        receiver.isClosed = true
        return nil
    }

    receiver.controller.Schedule()

}

func (receiver *UnixMsg) SendTo(addr string, fds ...int) error {
    if receiver.isClosed {
        return errors.ErrorDataSentIncomplete
    }
    return receiver.controller.SendFdTo(addr, fds...)
}

func (receiver *UnixMsg) Close() {
    if !receiver.isClosed {
        receiver.controller.Close()
        receiver.isClosed = true
    }
}
