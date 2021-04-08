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
	"github.com/packing/clove/codecs"
	"github.com/packing/clove/packets"
)

type UnixUDP struct {
	DataController
	Codec  *codecs.Codec
	Format *packets.PacketFormat
}

func CreateUnixUDPWithFormat(format *packets.PacketFormat, codec *codecs.Codec) *UnixUDP {
	s := new(UnixUDP)
	s.Codec = codec
	s.Format = format
	return s
}

func (receiver *UnixUDP) SetControllerAssociatedObject(o interface{}) {
}

func (receiver *UnixUDP) Bind(addr string) error {
	return nil
}

func (receiver UnixUDP) GetBindAddr() string {
	return ""
}

func (receiver *UnixUDP) SendTo(addr string, msgs ...codecs.IMData) ([]codecs.IMData, error) {
	return nil, nil
}

func (receiver *UnixUDP) SendFileHandler(addr string, fds ...int) error {
	return nil
}

func (receiver *UnixUDP) Close() {
}
