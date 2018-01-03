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
	"net/http"
	"bufio"
	"strings"
	"nbpy/codecs"
	"nbpy/bits"
)

const HttpHeaderMinLength = 16

type PacketParserHTTP struct {
}

type PacketPackagerHTTP struct {
}

func (receiver PacketParserHTTP) unEncrypt(in []byte) (error, []byte){
	return nil, in
}

func (receiver PacketParserHTTP) unCompress(in []byte, rawlen int) (error, []byte){
	return nil, in
}

func (receiver PacketParserHTTP) Prepare(*bytes.Buffer) (error, byte, byte, []byte) {
	return nil, codecs.ProtocolReserved, 0, nil
}

func (receiver PacketParserHTTP) TryParse(data *bytes.Buffer) (error,bool) {
	fB := bits.ReadAsciiCode(data.Bytes())
	if fB != 71 && fB != 80 {
		return ErrorDataNotMatch, false
	}
	if data.Len() < HttpHeaderMinLength {
		return ErrorDataNotReady, false
	}
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data.Bytes())))
	if err != nil {
		return err, false
	}

	if strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
		return ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserHTTP) Pop(raw *bytes.Buffer) (error, *Packet) {
	if raw.Len() < HttpHeaderMinLength {
		return ErrorDataNotReady, nil
	}
	return nil, nil
}

func (receiver PacketPackagerHTTP) Package(pck *Packet, raw []byte) (error, []byte) {
	var bs bytes.Buffer

	return nil, bs.Bytes()
}

var packetFormatHTTP = PacketFormat{Tag: "HTTP", Priority:0, Parser: PacketParserHTTP{}, Packager: PacketPackagerHTTP{}}
var PacketFormatHTTP = &packetFormatHTTP