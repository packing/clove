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
	"encoding/binary"
	"bytes"
	"nbpy/codecs"
	"fmt"
)

/*
packet struct {
mask 					-> bit (1)
flag 					-> bit (7)
protocol-type 			-> bit (4)
protocol-ver 			-> bit (4)
final-packet-size 		-> bit (24)
compress-data 			-> memory (final-packet-size - sizeof(packet-header))
}
*/
const (
	PacketNBHeaderLength 	= 5
	MaskNB 					= 0x80
	MaskNBCompressSupport 	= 0x1
	MaskNBEncrypt 			= 0x1 << 1
	MaskNBCompressed		= 0x1 << 2
	MaskNBReserved			= 0x1 << 3
	MaskNBFeature			= MaskNBReserved | MaskNB
)

type PacketParserNB struct {
}

type PacketPackagerNB struct {
}

func (receiver PacketParserNB) Prepare(*bytes.Buffer) (error, byte, byte, []byte) {
	return nil, codecs.ProtocolReserved, 0, nil
}

func (receiver PacketParserNB) TryParse(data *bytes.Buffer) (error,bool) {
	if data.Len() < PacketNBHeaderLength {
		return ErrorDataNotReady, false
	}

	peekData := data.Bytes()
	mask := peekData[0] & MaskNBFeature
	pLen := binary.BigEndian.Uint32(peekData[1:PacketNBHeaderLength])
	ptop := (pLen & 0xF0000000) >> 28
	ptov := (pLen & 0xF000000) >> 24
	pLen = pLen & PacketMaxLength

	if ptop == 0 || ptov == 0 {
		fmt.Println("1", ptop, ptov, pLen)
		return ErrorDataNotMatch, false
	}

	if pLen > PacketMaxLength || pLen < PacketNBHeaderLength {
		fmt.Println("2", pLen)
		return ErrorDataNotMatch, false
	}

	if mask != MaskNBFeature {
		fmt.Println("3", mask)
		return ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserNB) Pop(raw *bytes.Buffer) (error, *Packet) {
	if raw.Len() < PacketNBHeaderLength {
		return ErrorDataNotReady, nil
	}

	peekData 	:= raw.Bytes()
	opFlag 		:= peekData[0]
	mask 		:= opFlag & MaskNBFeature
	packetLen 	:= binary.BigEndian.Uint32(peekData[1:PacketNBHeaderLength])
	ptop 		:= byte((packetLen & 0xF0000000) >> 28)
	ptov 		:= byte((packetLen & 0xF000000) >> 24)
	packetLen 	= packetLen & PacketMaxLength

	if ptop == 0 || ptov == 0 {
		return ErrorDataIsDamage, nil
	}

	if packetLen > PacketMaxLength || packetLen < PacketNBHeaderLength {
		return ErrorDataIsDamage, nil
	}

	if mask != MaskNBFeature {
		return ErrorDataIsDamage, nil
	}

	if uint(packetLen) > uint(raw.Len()) {
		return ErrorDataNotReady, nil
	}

	packet := new(Packet)
	packet.Compressed 		= (opFlag & MaskNBCompressed) == MaskNBCompressed
	packet.Encrypted 		= (opFlag & MaskNBEncrypt) == MaskNBEncrypt
	packet.CompressSupport 	= (opFlag & MaskNBCompressSupport) == MaskNBCompressSupport
	packet.ProtocolType 	= ptop
	packet.ProtocolVer 		= ptov
	packet.Raw 				= peekData[PacketNBHeaderLength: packetLen]

	raw.Next(int(packetLen))

	return nil, packet
}

func (receiver PacketPackagerNB) Package(pck *Packet, raw []byte) (error, []byte) {
	header := make([]byte, PacketNBHeaderLength)

	var opFlag byte = 0
	if pck.Compressed {
		opFlag |= MaskNBCompressed
	}
	if pck.CompressSupport {
		opFlag |= MaskNBCompressSupport
	}
	if pck.Encrypted {
		opFlag |= MaskNBEncrypt
	}

	header[0] = MaskNBFeature | opFlag

	packetLen := uint32(len(raw)) + PacketNBHeaderLength
	ptoInfo := uint32((pck.ProtocolType << 4) | pck.ProtocolVer) << 24
	packetLen |= ptoInfo
	binary.BigEndian.PutUint32(header[1:], packetLen)


	return nil, bytes.Join([][]byte{header, raw}, []byte(""))
}

var packetFormatNB = PacketFormat{Tag: "NBPyPacket", Priority:-998, Parser: PacketParserNB{}, Packager: PacketPackagerNB{}}
var PacketFormatNB = &packetFormatNB