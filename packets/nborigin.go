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
	"nbpy/codecs"
	"nbpy/errors"
	"hash/crc32"
	"bytes"
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

type PacketParserNBOrigin struct {
}

type PacketPackagerNBOrigin struct {
}

func (receiver PacketParserNBOrigin) unEncrypt(in []byte) (error, []byte){
	return nil, in
}

func (receiver PacketParserNBOrigin) unCompress(in []byte, rawlen int) (error, []byte){
	return nil, in
}

func (receiver PacketParserNBOrigin) Prepare(*bytes.Buffer) (error, []byte) {
	return nil, nil
}

func (receiver PacketParserNBOrigin) TryParse(data *bytes.Buffer) (error,bool) {
	if data.Len() < PacketHeaderLength {
		return ErrorDataNotReady, false
	}

	sdata := data.Bytes()
	mask := binary.LittleEndian.Uint16(sdata)
	flag := binary.LittleEndian.Uint16(sdata[2:4])
	rlen := binary.LittleEndian.Uint32(sdata[4:8])
	plen := binary.LittleEndian.Uint32(sdata[8:12])
	if plen > PacketMaxLength || plen < PacketHeaderLength || plen < rlen {
		return ErrorDataNotMatch, false
	}

	if (flag & MaskCompress != MaskCompress) && (plen != (rlen + PacketHeaderLength)) {
		return ErrorDataNotMatch, false
	}

	if mask == 0xffff {
		return ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserNBOrigin) Pop(raw *bytes.Buffer) (error, *Packet) {
	if raw.Len() < PacketHeaderLength {
		return ErrorDataNotReady, nil
	}

	packetlen := int(binary.LittleEndian.Uint32(raw.Bytes()[8:12]))
	if packetlen > raw.Len() {
		return ErrorDataNotReady, nil
	}

	data := make([]byte, packetlen)
	readlen, err := raw.Read(data)
	if err != nil {
		return errors.Errorf("Buffer error> %s", err.Error()), nil
	}
	if readlen != packetlen {
		return ErrorDataIsDamage, nil
	}
	packet := new(Packet)
	packet.Mask = binary.LittleEndian.Uint16(data)
	flag := binary.LittleEndian.Uint16(data[2:4])
	packet.Compressed = (flag & MaskCompress) == MaskCompress
	packet.Encrypted = (flag & MaskEncrypt) == MaskEncrypt
	switch {
	case (flag & MaskImData) == MaskImData:
		packet.ProtocolType = codecs.ProtocolIM
	case (flag & MaskJson) == MaskJson:
		packet.ProtocolType = codecs.ProtocolJSON
	default:
		packet.ProtocolType = codecs.ProtocolMemory
	}
	packet.ProtocolVer = 1
	packet.CompressSupport = (flag & MaskCompressSupport) == MaskCompressSupport

	rdata := data[PacketHeaderLength:]
	if packet.Compressed {
		//todo:uncompress
		rawlen := int(binary.LittleEndian.Uint32(data[4:8]))
		err, rdata = receiver.unCompress(rdata, rawlen)
	}
	if packet.Encrypted {
		//todo:unencrypt
		err, rdata = receiver.unEncrypt(rdata)
	}

	packet.Raw = rdata

	return nil, packet
}

func (receiver PacketPackagerNBOrigin) Package(pck *Packet, raw []byte) (error, []byte) {
	data := make([]byte, PacketHeaderLength)
	binary.LittleEndian.PutUint16(data, pck.Mask)
	var flag uint16 = 0

	if pck.Compressed {
		flag |= MaskCompress
	}
	if pck.CompressSupport {
		flag |= MaskCompressSupport
	}
	if pck.Encrypted {
		flag |= MaskEncrypt
	}
	if pck.ProtocolType == codecs.ProtocolIM {
		flag |= MaskImData
	}
	if pck.ProtocolType == codecs.ProtocolJSON {
		flag |= MaskJson
	}
	binary.LittleEndian.PutUint16(data[2:4], flag)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(raw)))
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(raw)) + PacketHeaderLength)

	ieee := crc32.NewIEEE()
	ieee.Write(raw)
	binary.LittleEndian.PutUint32(data[12:16], ieee.Sum32())

	var bs bytes.Buffer
	bs.Write(data)
	bs.Write(raw)

	return nil, bs.Bytes()
}

var packetFormatNBOrigin = PacketFormat{Tag: "NBPyPacketOrigin", Priority:-999, Parser: PacketParserNBOrigin{}, Packager: PacketPackagerNBOrigin{}}
var PacketFormatNBOrigin = &packetFormatNBOrigin