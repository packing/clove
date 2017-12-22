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

package connections

import (
	"nbpy/thirdpartys/errors"
	"net"
	"bytes"
	"nbpy/codecs"
	"nbpy/utils"
	"nbpy/packets"
)

type ReceiveAction struct {
	err error
	data *bytes.Buffer
}

type Connection struct {
	Handle net.Conn
	Id int
	Encrypt bool
	Compress bool
	OnReceive func(peer Connection, msg codecs.IMData) error
	recvch chan ReceiveAction
	protocol uint32
	collectionCodecs map[uint32] *codecs.Codec
}

func (receiver *Connection) RegisterCodec(codec *codecs.Codec) (error) {
	k := uint32(codec.Protocol) << 16 | uint32(codec.Version)
	if receiver.collectionCodecs == nil {
		receiver.collectionCodecs = make(map[uint32] *codecs.Codec)
	}

	_, ok := receiver.collectionCodecs[k]
	if ok {
		return errors.Errorf("The codec protocol %d and version %d is exists.", codec.Protocol, codec.Version)
	}

	receiver.collectionCodecs[k] = codec
	return nil
}

func (receiver *Connection) FindCodec(protocol uint16, version uint16) (error, *codecs.Codec) {
	k := uint32(protocol) << 16 | uint32(version)
	if receiver.collectionCodecs == nil {
		receiver.collectionCodecs = make(map[uint32] *codecs.Codec)
	}
	t, ok := receiver.collectionCodecs[k]
	if ok {
		return nil, t
	}
	return errors.Errorf("Codec protocol %d and version %d is not exists.", protocol, version), nil
}

func (receiver *Connection) SetProtocol(tp, ver uint16) {
	receiver.protocol = uint32(tp) << 16 | uint32(ver)
}

func (receiver Connection) Welcome() {
	utils.LogVerbose(">>> 建立客户端连接 id: 0x%X addr: %s", receiver.Id, receiver.Handle.RemoteAddr().String())
}

func (receiver Connection) Bye() {
	utils.LogVerbose(">>> 关闭客户端连接 id: 0x%X addr: %s", receiver.Id, receiver.Handle.RemoteAddr().String())
}


func (receiver Connection) ParsePacket() error{
	utils.LogVerbose(">>> 进入解包尝试协程 (0x%X)", receiver.Id)

	//如果没有设定数据到达回调，则直接退出处理
	if receiver.OnReceive == nil {
		return errors.Errorf("OnReceive is not exists")
	}

	for {
		action := <- receiver.recvch
		parser := packets.PacketParser{Raw: action.data}
		for {
			err, packet := parser.Pop()
			if err != nil {
				if err.Error() == "Data length is not match" {
					utils.LogError(err.Error())
					goto exitLabel
				}
				break
			}

			if packet == nil {
				break
			}

			//动态决定当前通信协议类型和版本
			receiver.SetProtocol(packet.ProtocolType, packet.ProtocolVer)

			packetData := packet.Raw
			//如果是直接内存流数据协议，则直接转出至回调
			if packet.ProtocolType == codecs.ProtocolMemory {
				err := receiver.OnReceive(receiver, packetData)
				if err != nil {
					utils.LogError(err.Error())
					goto exitLabel
				}
				continue
			}

		readLabel:
			//寻找解码器
			err, codec := receiver.FindCodec(packet.ProtocolType, packet.ProtocolVer)
			if err != nil {
				return err
			}

			//开始使用解码器进行消息解码(单个封包可以包含多个消息体)
			err, msg, remianData := codec.Decoder.Decode(packetData)
			if err == nil {
				err := receiver.OnReceive(receiver, msg)
				if err != nil {
					break
				}
				packetData = remianData
				goto readLabel
			}
		}

		if action.err != nil{
			break
		}
	}

	exitLabel:
		utils.LogVerbose("<<< 退出解包尝试协程 (0x%X)", receiver.Id)
	return nil
}

func (receiver Connection) Lookup(interval int) error{
	utils.LogVerbose(">>> 进入数据通信处理协程 (0x%X)", receiver.Id)
	var recvbuffer bytes.Buffer

	receiver.recvch = make(chan ReceiveAction)
	go receiver.ParsePacket()

	for {
		var b = make([]byte, 1024)
		n, err := receiver.Handle.Read(b)
		if n > 0 {
			recvbuffer.Write(b[:n])
			receiver.recvch <- ReceiveAction{err, &recvbuffer}
		}

		if err != nil {
			receiver.recvch <- ReceiveAction{err, &recvbuffer}
			return errors.Errorf("Error at fd.Read.")
		}
	}
	utils.LogVerbose("<<< 退出数据通信处理协程 (0x%X)", receiver.Id)
	return nil
}

func (receiver Connection) Send(msgs ...codecs.IMData) (error) {
	packet := packets.Packet{
		Mask: 0,
		Encrypted: false,
		Compressed: false,
		CompressSupport: false,
	}
	packet.ProtocolType = uint16(receiver.protocol >> 16)
	packet.ProtocolVer = uint16(receiver.protocol << 16 >> 16)
	packager := packets.PacketPackager{Pck: &packet}
	for _, msg := range msgs {
		err, data := codecs.EncoderIMv1{}.Encode(&msg)
		if err == nil {
			packager.Push(data)
		}
	}

	err, data := packager.Package()
	if err != nil {
		return err
	}
	_, err = receiver.Handle.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (receiver Connection) SendPacket(packet packets.Packet) (error) {
	packager := packets.PacketPackager{Pck: &packet}
	err, data := packager.Package()
	if err != nil {
		return err
	}
	_, err = receiver.Handle.Write(data)
	if err != nil {
		return err
	}
	return nil
}