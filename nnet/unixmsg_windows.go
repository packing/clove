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
	"net"
)

type UnixMsg struct {
}

func CreateUnixMsg() *UnixMsg {
	s := new(UnixMsg)
	return s
}

func CreateUnixMsgWithBufferSize(bufWriteSize int, bufReadSize int) *UnixMsg {
	return CreateUnixMsg()
}

func (receiver *UnixMsg) SetControllerAssociatedObject(o interface{}) {}
func (receiver *UnixMsg) Bind(addr string) error                      { return nil }
func (receiver *UnixMsg) processClient(conn net.UnixConn)             {}
func (receiver *UnixMsg) SendTo(addr string, fds ...int) error        { return nil }
func (receiver *UnixMsg) Close()                                      {}
