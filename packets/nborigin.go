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
	"../codecs"
	"hash/crc32"
	"bytes"
	"../errors"
)

/*
packet struct {
mask 					-> unsigned short (2)
flag 					-> unsigned short (2)
uncompress-data-size 	-> int (4)
final-packet-size 		-> int (4)
verify-code 			-> unsigned long (4)
compress-data 			-> memory (final-packet-size - sizeof(packet-header))
}
*/

const (
	PacketNBOriginHeaderLength = 2 + 2 + 4 + 4 + 4
	MaskOriginCompress 			= 0x1
	MaskOriginEncrypt 			= 0x1 << 1
	MaskOriginImData 			= 0x1 << 2
	MaskOriginCompressSupport	= 0x1 << 3
	MaskOriginJson				= 0x1 << 4
)

type PacketParserNBOrigin struct {
}

type PacketPackagerNBOrigin struct {
}

func (receiver PacketParserNBOrigin) Prepare(in []byte) (error, int, byte, byte, []byte) {
	return nil, 0, codecs.ProtocolIM, 1, nil
}

func (receiver PacketParserNBOrigin) TryParse(in []byte) (error,bool) {
	if len(in) < PacketNBOriginHeaderLength {
		return errors.ErrorDataNotReady, false
	}

	sdata := in
	mask := binary.LittleEndian.Uint16(sdata)
	flag := binary.LittleEndian.Uint16(sdata[2:4])
	rlen := binary.LittleEndian.Uint32(sdata[4:8])
	plen := binary.LittleEndian.Uint32(sdata[8:12])
	if plen > PacketMaxLength || plen < PacketNBOriginHeaderLength || plen < rlen {
		return errors.ErrorDataNotMatch, false
	}

	if (flag & MaskOriginCompress != MaskOriginCompress) && (plen != (rlen + PacketNBOriginHeaderLength)) {
		return errors.ErrorDataNotMatch, false
	}

	if mask != 0 {
		return errors.ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserNBOrigin) Pop(in []byte) (error, *Packet, int) {
	if len(in) < PacketNBOriginHeaderLength {
		return errors.ErrorDataNotReady, nil, 0
	}

	packetlen := int(binary.LittleEndian.Uint32(in[8:12]))
	if packetlen > len(in) {
		return errors.ErrorDataNotReady, nil, 0
	}

	if packetlen < PacketNBOriginHeaderLength {
		return errors.ErrorDataIsDamage, nil, 0
	}

	data := in[:packetlen]

	packet := new(Packet)
	flag := binary.LittleEndian.Uint16(data[2:4])
	packet.Compressed = (flag & MaskOriginCompress) == MaskOriginCompress
	packet.Encrypted = (flag & MaskOriginEncrypt) == MaskOriginEncrypt
	switch {
	case (flag & MaskOriginImData) == MaskOriginImData:
		packet.ProtocolType = codecs.ProtocolIM
	case (flag & MaskOriginJson) == MaskOriginJson:
		packet.ProtocolType = codecs.ProtocolJSON
	default:
		packet.ProtocolType = codecs.ProtocolMemory
	}
	packet.ProtocolVer = 1
	packet.CompressSupport = (flag & MaskOriginCompressSupport) == MaskOriginCompressSupport

	if len(data) >= PacketNBOriginHeaderLength {
		packet.Raw = data[PacketNBOriginHeaderLength:]
	}else{
		packet.Raw = []byte("")
	}

	return nil, packet, packetlen
}

func (receiver PacketPackagerNBOrigin) Package(pck *Packet, raw []byte) (error, []byte) {
	data := make([]byte, PacketNBOriginHeaderLength)
	binary.LittleEndian.PutUint16(data, 0x0)
	var flag uint16 = 0

	if pck.Compressed {
		flag |= MaskOriginCompress
	}
	if pck.CompressSupport {
		flag |= MaskOriginCompressSupport
	}
	if pck.Encrypted {
		flag |= MaskOriginEncrypt
	}
	if pck.ProtocolType == codecs.ProtocolIM {
		flag |= MaskOriginImData
	}
	if pck.ProtocolType == codecs.ProtocolJSON {
		flag |= MaskOriginJson
	}
	binary.LittleEndian.PutUint16(data[2:4], flag)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(raw)))
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(raw)) +PacketNBOriginHeaderLength)


	ieee := crc32.NewIEEE()
	ieee.Write(raw)
	binary.LittleEndian.PutUint32(data[12:16], ieee.Sum32())

	return nil, bytes.Join([][]byte{data, raw}, []byte(""))
}

var packetFormatNBOrigin = PacketFormat{Tag: "NBPyPacketOrigin", Priority:-999, UnixNeed:true, Parser: PacketParserNBOrigin{}, Packager: PacketPackagerNBOrigin{}}
var PacketFormatNBOrigin = &packetFormatNBOrigin