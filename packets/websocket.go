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
	"fmt"
	"bytes"
	"bufio"
	"strings"

	"net/http"
	"encoding/base64"
	"encoding/binary"
	"crypto/sha1"

	"../codecs"
	"../bits"
	"../utils"
	"../errors"
)

const WSHeaderMinLength = 16
const WSDataMinLength = 2
const WSMagicStr = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
const WSRespFmt = "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\nSec-WebSocket-Protocol: %s\r\n\r\n"
const WSRespFmtWithoutProtocol = "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n"


var webSocketOrigin = []string{""}

type PacketParserWS struct {
}

type PacketPackagerWS struct {
}

func (receiver PacketParserWS) Prepare(in []byte) (error, int, byte, byte, []byte) {
	//utils.LogInfo(">>> HTTPHEADER > %s", string(in))
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(in)))
	if err != nil {
		return err, 0, codecs.ProtocolReserved, 0, nil
	}


	key := req.Header.Get("Sec-WebSocket-Key")
	pto := req.Header.Get("Sec-WebSocket-Protocol")

	//在此就回发握手响应
	accept := key + WSMagicStr
	hashcode := sha1.Sum([]byte(accept))
	fk := base64.StdEncoding.EncodeToString(hashcode[:])

    resp := fmt.Sprintf(WSRespFmtWithoutProtocol, fk)
	if pto != "" {
        resp = fmt.Sprintf(WSRespFmt, fk, pto)
    }

	var pton byte = codecs.ProtocolReserved
	var ptov byte = 0

	switch pto {
	case "nbpyimv1":
		pton = codecs.ProtocolIM
		ptov = 1
		utils.LogInfo(">>> 该连接数据协议为 Intermediate V1")
	case "nbpyimv2":
		pton = codecs.ProtocolIM
		ptov = 2
		utils.LogInfo(">>> 该连接数据协议为 Intermediate V2")
	case "nbpyjson":
		pton = codecs.ProtocolJSON
		ptov = 1
		utils.LogInfo(">>> 该连接数据协议为 JSON V1")
	case "json":
		pton = codecs.ProtocolJSON
		ptov = 1
		utils.LogInfo(">>> 该连接数据协议为 JSON V1")
	default:
		pton = codecs.ProtocolReserved
		ptov = 0
		utils.LogInfo(">>> 该连接数据协议为 未知")
	}

	utils.LogInfo(">>> 收到Websocket升级请求，允许升级连接")

	return nil, len(in), pton, ptov, []byte(resp)
}

func (receiver PacketParserWS) TryParse(in []byte) (error,bool) {
	fB := bits.ReadAsciiCode(in)
	if fB != 71 && fB != 80 {
		return errors.ErrorDataNotMatch, false
	}
	if len(in) < WSHeaderMinLength {
		return errors.ErrorDataNotReady, false
	}

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(in)))
	if err != nil {
		return err, false
	}

	ori := req.Header.Get("Origin")
	key := req.Header.Get("Sec-WebSocket-Key")
	ver := req.Header.Get("Sec-WebSocket-Version")
	upg := req.Header.Get("Upgrade")
	cnn := req.Header.Get("Connection")
	if strings.ToLower(upg) != "websocket" || cnn != "Upgrade" || key == "" || ori == "" || ver == "" {
		return errors.ErrorDataNotMatch, false
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
			return errors.ErrorDataNotMatch, false
		}
	}

	return nil, true
}

func (receiver PacketParserWS) Pop(in []byte) (error, *Packet, int) {
	if len(in) < WSDataMinLength {
		return errors.ErrorDataNotReady, nil, 0
	}

	peekData := in
	opCode := peekData[0] & 0xF

	switch opCode {
	case 0x8:
		return errors.ErrorRemoteReqClose, nil, 0
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

	if len(in) < headlen {
		return errors.ErrorDataNotReady, nil, 0
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
	if len(in) < totalLen {
		return errors.ErrorDataNotReady, nil, 0
	}

	payloadData := in[:totalLen]
	payloadData = payloadData[headlen:]

	if maskFlag == 1 {
		for i := 0; i < len(payloadData); i++ {
			payloadData[i] = payloadData[i] ^ mask[i%4]
		}
	}

	pck := new(Packet)
	pck.Raw = payloadData
	pck.Encrypted = false
	pck.Compressed = false
	pck.CompressSupport = false

	return nil, pck, totalLen
}

func (receiver PacketPackagerWS) Package(pck *Packet, raw []byte) (error, []byte) {
	rawLen := len(raw)
	header := make([]byte, 10)

	if pck.ProtocolType == codecs.ProtocolJSON {
		header[0] = byte(0x81)
	}else{
		header[0] = byte(0x82)
	}

	switch {
	case rawLen <= 125:
		header[1] = byte(rawLen)
		header = header[:2]
	case rawLen <= 0xFFFF:
		header[1] = 0x7e
		binary.BigEndian.PutUint16(header[2:4], uint16(rawLen))
		header = header[:4]
	default:
		header[1] = 0x7f
		binary.BigEndian.PutUint64(header[2:10], uint64(rawLen))
		header = header[:10]
	}

	finalBytes := bytes.Join([][]byte{header, raw}, []byte(""))

	return nil, finalBytes
}

func RegisterWebSocketOrigin(ori string) {
	webSocketOrigin = append(webSocketOrigin, ori)
}

var packetFormatWS = PacketFormat{Tag: "WebSocket", Priority:1, Parser: PacketParserWS{}, Packager: PacketPackagerWS{}}
var PacketFormatWS = &packetFormatWS