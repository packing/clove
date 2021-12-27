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
	"encoding/binary"

	"github.com/packing/clove/codecs"
	"github.com/packing/clove/errors"
	"github.com/packing/clove/utils"
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
	PacketNBHeaderLength  = 5
	MaskNB                = 0x80
	MaskNBCompressSupport = 0x1
	MaskNBEncrypt         = 0x1 << 1
	MaskNBCompressed      = 0x1 << 2
	MaskNBReserved        = 0x1 << 3
	MaskNBFeature         = MaskNBReserved | MaskNB
)

type PacketParserNB struct {
}

type PacketPackagerNB struct {
}

func (receiver PacketParserNB) Prepare(in []byte) (error, int, byte, byte, []byte) {
	return nil, 0, codecs.ProtocolIM, 2, nil
}

func (receiver PacketParserNB) TryParse(in []byte) (error, bool) {
	defer func() {
		utils.LogPanic(recover())
	}()
	if len(in) < PacketNBHeaderLength {
		return errors.ErrorDataNotReady, false
	}

	peekData := in
	mask := peekData[0] & MaskNBFeature
	pLen := binary.BigEndian.Uint32(peekData[1:PacketNBHeaderLength])
	ptop := (pLen & 0xF0000000) >> 28
	ptov := (pLen & 0xF000000) >> 24
	pLen = pLen & PacketMaxLength

	if ptop == 0 || ptov == 0 {
		return errors.ErrorDataNotMatch, false
	}

	if pLen > PacketMaxLength || pLen < PacketNBHeaderLength {
		return errors.ErrorDataNotMatch, false
	}

	if mask != MaskNBFeature {
		return errors.ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserNB) Pop(in []byte) (error, *Packet, int) {
	defer func() {
		utils.LogPanic(recover())
	}()
	if len(in) < PacketNBHeaderLength {
		return errors.ErrorDataNotReady, nil, 0
	}

	peekData := in
	opFlag := peekData[0]
	mask := opFlag & MaskNBFeature
	packetLen := binary.BigEndian.Uint32(peekData[1:PacketNBHeaderLength])
	ptop := byte((packetLen & 0xF0000000) >> 28)
	ptov := byte((packetLen & 0xF000000) >> 24)
	packetLen = packetLen & PacketMaxLength

	if ptop == 0 || ptov == 0 {
		return errors.ErrorDataIsDamage, nil, 0
	}

	if packetLen > PacketMaxLength || packetLen < PacketNBHeaderLength {
		return errors.ErrorDataIsDamage, nil, 0
	}

	if mask != MaskNBFeature {
		return errors.ErrorDataIsDamage, nil, 0
	}

	if uint(packetLen) > uint(len(in)) {
		return errors.ErrorDataNotReady, nil, int(packetLen)
	}

	packet := new(Packet)
	packet.Compressed = (opFlag & MaskNBCompressed) == MaskNBCompressed
	packet.Encrypted = (opFlag & MaskNBEncrypt) == MaskNBEncrypt
	packet.CompressSupport = (opFlag & MaskNBCompressSupport) == MaskNBCompressSupport
	packet.ProtocolType = ptop
	packet.ProtocolVer = ptov
	packet.Raw = peekData[PacketNBHeaderLength:packetLen]

	return nil, packet, int(packetLen)
}

func (receiver PacketPackagerNB) Package(pck *Packet, raw []byte) (error, []byte) {
	defer func() {
		utils.LogPanic(recover())
	}()
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
	ptoInfo := uint32((pck.ProtocolType<<4)|pck.ProtocolVer) << 24
	packetLen |= ptoInfo
	binary.BigEndian.PutUint32(header[1:], packetLen)

	return nil, bytes.Join([][]byte{header, raw}, []byte(""))
}

var packetFormatNB = PacketFormat{Tag: "NBPyPacket", Priority: -998, UnixNeed: true, Parser: PacketParserNB{}, Packager: PacketPackagerNB{}}
var PacketFormatNB = &packetFormatNB
