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

type UnixController struct {
    OnStop     OnControllerStop
    DataRW     *DataReadWriter
}

func createUnixController(ioSrc net.UnixConn, dataRW *DataReadWriter) *UnixController {
    sor := new(UnixController)
    return sor
}

func (receiver *UnixController) SetTag(tag int) {}

func (receiver *UnixController) GetTag() int { return 0 }

func (receiver *UnixController) SetAssociatedObject(o interface{}) { }

func (receiver *UnixController) GetAssociatedObject() interface{} { return nil }

func (receiver UnixController) GetSource() string { return "" }

func (receiver UnixController) GetSessionID() SessionID { return SessionID(0) }

func (receiver UnixController) Close() { }

func (receiver UnixController) Discard() { }

func (receiver UnixController) CloseOnSended() { }

func (receiver UnixController) Read(l int) ([]byte, int) { return nil, 0 }

func (receiver UnixController) Peek(l int) ([]byte, int) { return nil, 0 }

func (receiver UnixController) Write(data []byte) {}

func (receiver UnixController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) { return msg, errors.ErrorDataSentIncomplete }

func (receiver UnixController) ReadFrom() (string, []byte, int) { return "", nil, 0 }

func (receiver *UnixController) WriteTo(addr string, data []byte) { }

func (receiver *UnixController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) { return nil, nil }

func (receiver UnixController) SendFdTo(addr string, fds ...int) error { return nil }

func (receiver *UnixController) Schedule() { }
