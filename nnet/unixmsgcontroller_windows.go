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
    "net"
)

type UnixMsgController struct {
    OnStop  OnControllerStop
}

func createUnixMsgController(ioSrc net.UnixConn) *UnixMsgController {
    sor := new(UnixMsgController)
    return sor
}

func (receiver *UnixMsgController) SetAssociatedObject(o interface{}) {
}

func (receiver *UnixMsgController) GetAssociatedObject() interface{} {
    return nil
}

func (receiver *UnixMsgController) SetTag(tag int) {
}

func (receiver *UnixMsgController) GetTag() int {
    return 0
}

func (receiver UnixMsgController) GetSource() string {
    return ""
}

func (receiver UnixMsgController) GetSessionID() SessionID {
    return SessionID(0)
}

func (receiver UnixMsgController) Close() {
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

func (receiver *UnixMsgController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) {
    return nil, nil
}

func (receiver UnixMsgController) SendFdTo(addr string, fds ...int) error { return nil }

func (receiver *UnixMsgController) Schedule() { }
