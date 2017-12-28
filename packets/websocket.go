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
	"nbpy/utils"
	"strings"
	"encoding/base64"
	"crypto/sha1"
	"fmt"
	"encoding/binary"
)

const WSHeaderMinLength = 16
const WSDataMinLength = 2
const WSMagicStr = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
const WSRespFmt = "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n"


var webSocketOrigin = []string{""}

type PacketParserWS struct {
}

type PacketPackagerWS struct {
}

func (receiver PacketParserWS) unEncrypt(in []byte) (error, []byte){
	return nil, in
}

func (receiver PacketParserWS) unCompress(in []byte, rawlen int) (error, []byte){
	return nil, in
}

func (receiver PacketParserWS) Prepare(data *bytes.Buffer) (error, []byte) {

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data.Bytes())))
	if err != nil {
		return err, nil
	}

	key := req.Header.Get("Sec-WebSocket-Key")

	//重置读缓冲区
	data.Reset()

	//在此就回发握手响应
	//baccept := make([]byte, 64)
	accept := key + WSMagicStr
	hashcode := sha1.Sum([]byte(accept))
	fk := base64.StdEncoding.EncodeToString(hashcode[:])
	resp := fmt.Sprintf(WSRespFmt, fk)

	return nil, []byte(resp)
}

func (receiver PacketParserWS) TryParse(data *bytes.Buffer) (error,bool) {
	if data.Len() < WSHeaderMinLength {
		return ErrorDataNotReady, false
	}

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data.Bytes())))
	if err != nil {
		return err, false
	}

	ori := req.Header.Get("Origin")
	key := req.Header.Get("Sec-WebSocket-Key")
	ver := req.Header.Get("Sec-WebSocket-Version")
	upg := req.Header.Get("Upgrade")
	cnn := req.Header.Get("Connection")
	if strings.ToLower(upg) != "websocket" || cnn != "Upgrade" || key == "" || ori == "" || ver == "" {
		return ErrorDataNotMatch, false
	}

	if len(webSocketOrigin) > 1 {
		ballowed := false
		for _, o := range webSocketOrigin[1:] {
			if o == ori {
				ballowed = true
				break
			}
		}
		if !ballowed {
			utils.LogWarn("请求来源 %s 不被允许", ori)
			return ErrorDataNotMatch, false
		}
	}

	return nil, true
}

func (receiver PacketParserWS) Pop(raw *bytes.Buffer) (error, *Packet) {
	if raw.Len() < WSDataMinLength {
		return ErrorDataNotReady, nil
	}

	peekData := raw.Bytes()
	opCode := peekData[0] & 0xF

	switch opCode {
	case 0x8:
		return ErrorRemoteReqClose, nil
	default:

	}

	maskFlag := peekData[1] >> 7
	maskBits := 4
	if maskFlag != 1 {
		maskBits = 0
	}
	payloadLen := int(peekData[1] & 0x7F)
	headlen := WSDataMinLength
	dataLen := uint64(0)

	switch payloadLen {
	case 126:
		headlen = 4 + maskBits
	case 127:
		headlen = 10 + maskBits
	default:
		headlen = WSDataMinLength + maskBits
	}

	if raw.Len() < headlen {
		return ErrorDataNotReady, nil
	}

	mask := make([]byte, 4)
	switch payloadLen {
	case 126:
		dataLen = uint64(binary.BigEndian.Uint16(peekData[2:4]))
		if maskFlag == 1 {
			mask = peekData[4:8]
		}
	case 127:
		dataLen = binary.BigEndian.Uint64(peekData[2:10])
		if maskFlag == 1 {
			mask = peekData[10:14]
		}
	default:
		dataLen = uint64(payloadLen)
		if maskFlag == 1 {
			mask = peekData[2:6]
		}
	}

	totalLen := headlen + int(dataLen & 0xFFFFFFFF)
	if raw.Len() < totalLen {
		return ErrorDataNotReady, nil
	}

	payloadData := raw.Next(totalLen)
	payloadData = payloadData[headlen:]

	if maskFlag == 1 {
		for i := 0; i < len(payloadData); i++ {
			payloadData[i] = payloadData[i] ^ mask[i%4]
		}
	}

	return nil, nil
}

func (receiver PacketPackagerWS) Package(pck *Packet, raw []byte) (error, []byte) {
	var bs bytes.Buffer

	return nil, bs.Bytes()
}

func RegisterWebSocketOrigin(ori string) {
	webSocketOrigin = append(webSocketOrigin, ori)
}

var packetFormatWS = PacketFormat{Tag: "WebSocket", Priority:1, Parser: PacketParserWS{}, Packager: PacketPackagerWS{}}
var PacketFormatWS = &packetFormatWS