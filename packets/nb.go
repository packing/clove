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
	"nbpy/errors"
	"hash/crc32"
	"bytes"
)

/*
packet struct {
mask 					-> unsigned short (2)
flag 					-> unsigned short (2)
protocol-type 			-> int16 (2)
protocol-ver 			-> int16 (2)
final-packet-size 		-> int (4)
verify-code 			-> unsigned long (4)
compress-data 			-> memory (final-packet-size - sizeof(packet-header))
}
*/

type PacketParserNB struct {
}

type PacketPackagerNB struct {
}

func (receiver PacketParserNB) unEncrypt(in []byte) (error, []byte){
	return nil, in
}

func (receiver PacketParserNB) unCompress(in []byte, rawlen int) (error, []byte){
	return nil, in
}

func (receiver PacketParserNB) Prepare(*bytes.Buffer) (error, []byte) {
	return nil, nil
}

func (receiver PacketParserNB) TryParse(data *bytes.Buffer) (error,bool) {
	if data.Len() < PacketHeaderLength {
		return ErrorDataNotReady, false
	}

	sdata := data.Bytes()
	mask := binary.LittleEndian.Uint16(sdata)
	plen := binary.LittleEndian.Uint32(sdata[8:12])
	if plen > PacketMaxLength || plen < PacketHeaderLength {
		return ErrorDataNotMatch, false
	}

	if mask != 0xffff {
		return ErrorDataNotMatch, false
	}

	return nil, true
}

func (receiver PacketParserNB) Pop(raw *bytes.Buffer) (error, *Packet) {
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
	packet.ProtocolType = binary.LittleEndian.Uint16(data[4:6])
	packet.ProtocolVer = binary.LittleEndian.Uint16(data[6:8])
	packet.CompressSupport = (flag & MaskCompressSupport) == MaskCompressSupport

	data = data[PacketHeaderLength:]
	if packet.Compressed {
		//todo:uncompress
		err, data = receiver.unCompress(data, 0)
	}
	if packet.Encrypted {
		//todo:unencrypt
		err, data = receiver.unEncrypt(data)
	}

	packet.Raw = data

	return nil, packet
}

func (receiver PacketPackagerNB) Package(pck *Packet, raw []byte) (error, []byte) {
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
	binary.LittleEndian.PutUint16(data[2:4], flag)
	binary.LittleEndian.PutUint16(data[4:6], pck.ProtocolType)
	binary.LittleEndian.PutUint16(data[6:8], pck.ProtocolVer)
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(raw)) + PacketHeaderLength)

	ieee := crc32.NewIEEE()
	ieee.Write(raw)
	binary.LittleEndian.PutUint32(data[12:16], ieee.Sum32())

	var bs bytes.Buffer
	bs.Write(data)
	bs.Write(raw)

	return nil, bs.Bytes()
}

var packetFormatNB = PacketFormat{Tag: "NBPyPacket", Priority:-998, Parser: PacketParserNB{}, Packager: PacketPackagerNB{}}
var PacketFormatNB = &packetFormatNB