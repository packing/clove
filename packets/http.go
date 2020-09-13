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
	"net/http"
	"bufio"
	"strings"
	"../codecs"
	"../bits"
	"../errors"
	"bytes"
)

const HttpHeaderMinLength = 16

type PacketParserHTTP struct {
}

type PacketPackagerHTTP struct {
}

func (receiver PacketParserHTTP) Prepare(in []byte) (error, int, byte, byte, []byte) {
	return nil, 0, codecs.ProtocolIM, 2, nil
}

func (receiver PacketParserHTTP) TryParse(in []byte) (error,bool) {
	fB := bits.ReadAsciiCode(in)
	if fB != 71 && fB != 80 {
		return errors.ErrorDataNotMatch, false
	}
	if len(in) < HttpHeaderMinLength {
		return errors.ErrorDataNotReady, false
	}
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(in)))
	if err != nil {
		return err, false
	}

	if strings.ToLower(req.Header.Get("Upgrade")) == "websocket" {
		return errors.ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserHTTP) Pop(in []byte) (error, *Packet, int) {
	if len(in) < HttpHeaderMinLength {
		return errors.ErrorDataNotReady, nil, 0
	}

	dict := make(codecs.IMMap)
	dict["http"] = string(in[:])
	d := codecs.IMData(dict)
	err, out := codecs.CodecIMv2.Encoder.Encode(&d)
	if err != nil {
		return errors.ErrorDataNotReady, nil, 0
	}

	pck := new(Packet)
	pck.Raw = out
	pck.Encrypted = false
	pck.Compressed = false
	pck.CompressSupport = false
	pck.ProtocolType = codecs.ProtocolIM
	pck.ProtocolVer = 2

	return nil, pck, len(in)
}

func (receiver PacketPackagerHTTP) Package(pck *Packet, raw []byte) (error, []byte) {

	return nil, []byte("")
}

var packetFormatHTTP = PacketFormat{Tag: "HTTP", Priority:0, Parser: PacketParserHTTP{}, Packager: PacketPackagerHTTP{}}
var PacketFormatHTTP = &packetFormatHTTP