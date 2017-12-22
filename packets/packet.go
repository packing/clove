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
	"hash/crc32"
	"nbpy/codecs"
	"nbpy/thirdpartys/errorstack"
)

/*
old style packet struct {
mask 					-> unsigned short (2)
flag 					-> unsigned short (2)
uncompress-data-size 	-> int (4)
final-packet-size 		-> int (4)
verify-code 			-> unsigned long (4)
compress-data 			-> memory (final-packet-size - sizeof(packet-header))
}
*/

/*
new style packet struct {
mask 					-> unsigned short (2)
flag 					-> unsigned short (2)
protocol-type 			-> int16 (2)
protocol-ver 			-> int16 (2)
final-packet-size 		-> int (4)
verify-code 			-> unsigned long (4)
compress-data 			-> memory (final-packet-size - sizeof(packet-header))
}
*/

const (
	MaskNewStylePacket 	= 0xff
	PacketHeaderLength 	= 2 + 2 + 4 + 4 + 4
	MaskCompress 		= 0x1
	MaskEncrypt 		= 0x1 << 1
	MaskImData 			= 0x1 << 2
	MaskCompressSupport	= 0x1 << 3
	MaskJson			= 0x1 << 4
)

type Packet struct {
	Mask uint16
	Encrypted bool
	Compressed bool
	ProtocolType uint16
	ProtocolVer uint16
	CompressSupport bool
	Raw []byte
}

type PacketParser struct {
	Raw *bytes.Buffer
}

type PacketPackager struct {
	Pck *Packet
	Raw bytes.Buffer
}

func (receiver PacketParser) unEncrypt(in []byte) (error, []byte){
	return nil, in
}

func (receiver PacketParser) unCompress(in []byte, rawlen int) (error, []byte){
	return nil, in
}

func (receiver PacketParser) popOldStylePacket() (error, *Packet) {
	packetlen := int(binary.LittleEndian.Uint32(receiver.Raw.Bytes()[8:12]))
	if packetlen > receiver.Raw.Len() {
		return errorstack.Errorf("Data length is not enough"), nil
	}

	data := make([]byte, packetlen)
	readlen, err := receiver.Raw.Read(data)
	if err != nil {
		return errorstack.Errorf("Buffer error> %s", err.Error()), nil
	}
	if readlen != packetlen {
		return errorstack.Errorf("Data length is not match"), nil
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

	data = data[PacketHeaderLength:]
	if packet.Compressed {
		//todo:uncompress
		rawlen := int(binary.LittleEndian.Uint32(receiver.Raw.Bytes()[4:8]))
		err, data = receiver.unCompress(data, rawlen)
	}
	if packet.Encrypted {
		//todo:unencrypt
		err, data = receiver.unEncrypt(data)
	}

	packet.Raw = data

	return nil, packet
}

func (receiver PacketParser) popNewStylePacket() (error, *Packet) {
	packetlen := int(binary.LittleEndian.Uint32(receiver.Raw.Bytes()[8:12]))
	if packetlen > receiver.Raw.Len() {
		return errorstack.Errorf("Data length is not enough"), nil
	}

	data := make([]byte, packetlen)
	readlen, err := receiver.Raw.Read(data)
	if err != nil {
		return errorstack.Errorf("Buffer error> %s", err.Error()), nil
	}
	if readlen != packetlen {
		return errorstack.Errorf("Data length is not match"), nil
	}
	packet := new(Packet)
	packet.Mask = binary.LittleEndian.Uint16(data)
	flag := binary.LittleEndian.Uint16(data[2:4])
	packet.Compressed = (flag & MaskCompress) == MaskCompress
	packet.Encrypted = (flag & MaskEncrypt) == MaskEncrypt
	packet.ProtocolType = binary.LittleEndian.Uint16(receiver.Raw.Bytes()[4:6])
	packet.ProtocolVer = binary.LittleEndian.Uint16(receiver.Raw.Bytes()[6:8])
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

func (receiver PacketParser) Pop() (error, *Packet) {
	if receiver.Raw.Len() < PacketHeaderLength {
		return errorstack.Errorf("Header length is not enough"), nil
	}
	if binary.LittleEndian.Uint16(receiver.Raw.Bytes()) == MaskNewStylePacket {
		//使用新风格
		return receiver.popNewStylePacket()
	} else {
		//旧风格
		return receiver.popOldStylePacket()
	}
}

func (receiver *PacketPackager) Push(data []byte) (error) {
	_, err := receiver.Raw.Write(data)
	return err
}

func (receiver *PacketPackager) Package() (error, []byte) {
	data := make([]byte, PacketHeaderLength)
	binary.LittleEndian.PutUint16(data, receiver.Pck.Mask)
	var flag uint16 = 0
	if receiver.Pck.Mask == MaskNewStylePacket {
		//新风格
		//binary.LittleEndian.PutUint32(data[4:8], uint32(receiver.Raw.Len()))
	}else{
		//旧风格
		if receiver.Pck.Compressed {
			flag |= MaskCompress
		}
		if receiver.Pck.CompressSupport {
			flag |= MaskCompressSupport
		}
		if receiver.Pck.Encrypted {
			flag |= MaskEncrypt
		}
		if receiver.Pck.ProtocolType == codecs.ProtocolIM {
			flag |= MaskImData
		}
		if receiver.Pck.ProtocolType == codecs.ProtocolJSON {
			flag |= MaskJson
		}
		binary.LittleEndian.PutUint32(data[4:8], uint32(receiver.Raw.Len()))
		binary.LittleEndian.PutUint32(data[8:12], uint32(receiver.Raw.Len()) + PacketHeaderLength)
	}
	binary.LittleEndian.PutUint16(data[2:4], flag)

	ieee := crc32.NewIEEE()
	ieee.Write(receiver.Raw.Bytes())
	binary.LittleEndian.PutUint32(data[12:16], ieee.Sum32())

	var bs bytes.Buffer
	bs.Write(data)
	bs.Write(receiver.Raw.Bytes())

	return nil, bs.Bytes()
}

func GetDecoder(protocolType, protocolVer uint16, data []byte) (error, codecs.Decoder){
	switch protocolType {
	case codecs.ProtocolIM:
		switch protocolVer {
		case 1:
			return nil, codecs.DecoderIMv1{}
		default:
			return errorstack.Errorf("Protocol %d(%d) is not supported", protocolType, protocolVer), nil
		}
	case codecs.ProtocolJSON:
		switch protocolVer {
		case 1:
			return nil, codecs.DecoderJSONv1{}
		default:
			return errorstack.Errorf("Protocol %d(%d) is not supported", protocolType, protocolVer), nil
		}
	default:

	}

	return errorstack.Errorf("Protocol %d(%d) is not supported", protocolType, protocolVer), nil
}