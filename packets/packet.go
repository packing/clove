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

package packets

import (
	"bytes"
	"nbpy/errors"
)
const PacketMaxLength 	= 0xFFFFFF

var ErrorDataNotReady 	= errors.Errorf("Data length is not enough")
var ErrorDataNotMatch 	= errors.Errorf("Cannot match any packet format")
var ErrorDataIsDamage 	= errors.Errorf("Data length is not match")
var ErrorRemoteReqClose	= errors.Errorf("The remote host request close it")

type Packet struct {
	Encrypted bool
	Compressed bool
	ProtocolType byte
	ProtocolVer byte
	CompressSupport bool
	Raw []byte
}

type PacketParser interface {
	TryParse(*bytes.Buffer) (error, bool)
	Prepare(*bytes.Buffer) (error, byte, byte, []byte)
	Pop(*bytes.Buffer) (error, *Packet)
}

type PacketPackager interface {
	Package(*Packet, []byte) (error, []byte)
}


type PacketFormat struct {
	Tag string
	Priority int
	Parser PacketParser
	Packager PacketPackager
}