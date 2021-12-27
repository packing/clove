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

package nnet

import (
	"bytes"

	"github.com/packing/clove/codecs"
	"github.com/packing/clove/env"
	"github.com/packing/clove/errors"
	"github.com/packing/clove/packets"
	"github.com/packing/clove/utils"
)

type DataController struct {
	OnEncrypt     func([]byte) (error, []byte)
	OnDecrypt     func([]byte) (error, []byte)
	OnCompress    func([]byte) (error, []byte)
	OnUncompress  func([]byte) (error, []byte)
	OnDataDecoded func(Controller, string, codecs.IMData) error
}

type DataReadWriter struct {
	DataController
	codec           *codecs.Codec
	format          *packets.PacketFormat
	compressEnabled bool
	virgin          bool
}

func createDataReadWriter(codec *codecs.Codec, format *packets.PacketFormat) *DataReadWriter {
	s := new(DataReadWriter)
	s.codec = codec
	s.format = format
	s.virgin = true
	return s
}

func (receiver *DataReadWriter) PeekPacketLength(stream []byte) int {
	if receiver.format == nil {
		return 0
	}
	err, _, readLen := receiver.format.Parser.Pop(stream)
	if err != nil {
		if err == errors.ErrorDataNotReady {
			return readLen
		}
		return 0
	}
	return readLen
}

func (receiver *DataReadWriter) ReadStream(controller Controller, buf *utils.MutexBuffer) error {

	var inData, _ = buf.Peek(1024)

	//var peekData []byte
	//var nPeek int
	if receiver.format == nil {
		//如果没有指定封包格式，则进行封包格式选定操作
		//peekData = inData[:1024]
		err, pf := env.MatchPacketFormat(inData)
		if err != nil {
			if err == errors.ErrorDataNotMatch {
				//未能匹配任何封包格式，将会中断该连接
				utils.LogWarn("连接 %s 未能匹配到任何通信封包协议, 将会被强行关闭", controller.GetSource())
				return err
			} else {
				//可能数据不足，继续接收事件以等待数据完整
				return nil
			}
		}
		receiver.format = pf
	}

	if receiver.virgin {
		//如果仍处于起始状态，调用封包解包器的预处理方法进行某些握手操作并尝试明确协议类型(如果有需要的话，如websocket)
		//if peekData == nil {
		//    peekData = inData[:1024]
		//}
		receiver.virgin = false
		err, readLen, pto, ptov, sd := receiver.format.Parser.Prepare(inData)

		if receiver.codec == nil {
			if pto == codecs.ProtocolReserved && ptov == 0 && receiver.format == packets.PacketFormatWS {
				utils.LogWarn("连接 %s 没有指定编解码器类型, 将会使用系统默认类型 %s", controller.GetSource(), wsCodecDefault.Name)
				receiver.codec = wsCodecDefault
			} else {
				//寻找解码器
				err, codec := env.FindCodec(pto, ptov)
				if err != nil {
					//找不到对应到解码器，将会中断该连接
					utils.LogWarn("连接 %s 找不到对应解码器, 将会被强行关闭", controller.GetSource())
					return err
				}
				receiver.codec = codec
			}
		}

		if err == nil && sd != nil {
			//有待反馈数据，发送之，并将假定此类型连接等待反馈数据方会续发数据，如websocket，所以此处直接忽略剩余数据处理
			controller.Write(sd)
			controller.Discard()
			return nil
		}

		if readLen > 0 {
			buf.Next(readLen)
			inData, _ = buf.Peek(1024)
		}
		if buf.Len() == 0 {
			//如果数据已经读完, 等待后续数据到达
			return nil
		}
	}

dataCtrl:
	for {
		//peekData = inData[:1024*1024]
		inData, peekLen := buf.Peek(1024)
		pl := receiver.PeekPacketLength(inData)
		if pl == 0 {
			utils.LogError("!!! 封包解包失败，连接 %s 将被关闭1", controller.GetSource())
			return errors.ErrorDataNotMatch
		}
		if pl == -1 {
			break dataCtrl
		} else {
		}

		inData, peekLen = buf.Peek(pl)
		if pl > peekLen {
			//utils.LogError("!!! 封包解包失败，连接 %s 将被关闭2", controller.GetSource())
			break dataCtrl
		}

		err, packet, readLen := receiver.format.Parser.Pop(inData)
		if err != nil {
			if err != errors.ErrorDataNotReady {
				utils.LogError("!!! 封包解包失败，连接 %s 将被关闭3", controller.GetSource())
				return err
			}
			utils.LogError("!!! 解不出来包啊这是什么狗屁数据啊大哥", len(inData))
			break dataCtrl
		}

		if packet == nil {
			utils.LogError("!!! 封包解包失败，连接 %s 将被关闭4", controller.GetSource())
			return errors.ErrorDataNotMatch
		}

		if receiver.codec == nil {
			//如果当前连接未确定通信协议，根据当前封包属性决定通信协议类型和版本
			//寻找解码器
			err, codec := env.FindCodec(packet.ProtocolType, packet.ProtocolVer)
			if err != nil {
				//找不到对应到解码器，将会中断该连接
				utils.LogWarn("找不到对应解码器, 连接 %s 将会被强行关闭", controller.GetSource())
				return err
			}

			if receiver.codec == nil {
				//如果并不是直接内存数据流，而编解码器又未能就绪，则直接中断该连接
				utils.LogError("编解码器未能就绪, 连接 %s 将会被强行关闭", controller.GetSource())
				return errors.ErrorCodecNotReady
			}
			receiver.codec = codec
		}

		//根据对端封包标识标明对端是否支持压缩
		if packet.CompressSupport {
			receiver.compressEnabled = packet.CompressSupport
		}

		//controller.Read(readLen)
		buf.Next(readLen)

		packetData := packet.Raw

		//解密处理
		if packet.Encrypted {
			if receiver.OnDecrypt != nil {
				err, deEncryptData := receiver.OnDecrypt(packetData)
				if err == nil {
					packetData = deEncryptData
				} else {
					utils.LogWarn("进行数据解密失败, 连接 %s 将会被强行关闭", controller.GetSource())
					return errors.ErrorDecryptFunctionNotBind
				}
			} else {
				utils.LogWarn("连接 %s 未绑定解密函数, 将会被强行关闭", controller.GetSource())
				return errors.ErrorDecryptFunctionNotBind
			}
		}

		if packet.Compressed {
			if receiver.OnUncompress != nil {
				err, rawData := receiver.OnUncompress(packetData)
				if err == nil {
					packetData = rawData
				} else {
					utils.LogWarn("进行数据解压缩失败, 连接 %s 将会被强行关闭", controller.GetSource())
					return errors.ErrorUncompressFunctionNotBind
				}
			} else {
				utils.LogWarn("连接 %s 未绑定解压缩函数, 将会被强行关闭", controller.GetSource())
				return errors.ErrorUncompressFunctionNotBind
			}
		}

	dataDecode:
		//开始使用解码器进行消息解码(单个封包允许包含多个消息体，所以此处有label供goto回流继续解码下一块消息体)
		err, msg, remianData := receiver.codec.Decoder.Decode(packetData)
		if err == nil {
			if receiver.OnDataDecoded != nil {
				IncDecodeInstanceCount()
				err := receiver.OnDataDecoded(controller, controller.GetSource(), msg)
				DecDecodeInstanceCount()
				if err != nil {
					utils.LogError("逻辑处理返回错误 > %s, 连接 %s 将会被强行关闭.数据长度: %d", err.Error(), controller.GetSource(), len(packetData))
					return err
				} else {
				}
			}
			packetData = remianData
			if len(packetData) > 0 {
				goto dataDecode
			}
		} else if err != errors.ErrorDataNotEnough {
			utils.LogInfo("Err: ", err)
			utils.LogInfo("Raw: ", packetData)
			utils.LogInfo("readLen: %d", readLen)
			utils.LogWarn("进行数据解码失败, 连接 %s 将会被强行关闭", controller.GetSource())
			return err
		} else {
			utils.LogError("!!! 我他妈没解出来大哥我也不知道为什么", err)
			/*
				if receiver.OnDataDecoded != nil {
					err := receiver.OnDataDecoded(controller, controller.GetSource(), []byte(""))
					if err != nil {
						utils.LogError("逻辑处理返回错误 > %s, 连接 0x%X 将会被强行关闭",err.Error(), controller.GetSource())
						return err
					}
				}*/
		}
	}
	return nil
}

func (receiver *DataReadWriter) PackStream(controller Controller, msgs ...codecs.IMData) ([]byte, []codecs.IMData, error) {
	if receiver.format == nil {
		utils.LogWarn("!!! 发送未编码数据失败，连接 %s 封包解包器未就绪", controller.GetSource())
		return []byte(""), msgs, errors.ErrorPacketFormatNotReady
	}
	if receiver.codec == nil {
		utils.LogWarn("!!! 发送未编码数据失败，连接 %s 编解码器未就绪", controller.GetSource())
		return []byte(""), msgs, errors.ErrorCodecNotReady
	}

	var errorMsgs []codecs.IMData = nil
	encodeDatas := make([][]byte, 0)

	for i, msg := range msgs {
		err, data := receiver.codec.Encoder.Encode(&msg)
		if err == nil {
			encodeDatas = append(encodeDatas, data)
		} else {
			errorMsgs = msgs[i:]
			break
		}
	}

	finalData := bytes.Join(encodeDatas, []byte(""))

	packet := packets.Packet{
		Encrypted:       false,
		Compressed:      false,
		CompressSupport: false,
	}
	packet.ProtocolType = receiver.codec.Protocol
	packet.ProtocolVer = receiver.codec.Version

	if receiver.OnEncrypt != nil {
		err, encryptData := receiver.OnEncrypt(finalData)
		if err == nil {
			finalData = encryptData
			packet.Encrypted = true
		}
	}

	if receiver.compressEnabled && (receiver.OnCompress != nil) {
		err, compressData := receiver.OnCompress(finalData)
		if err == nil {
			finalData = compressData
			packet.Compressed = true
		}
	}

	err, data := receiver.format.Packager.Package(&packet, finalData)
	if err != nil {
		return []byte(""), msgs, err
	}

	return data, errorMsgs, err
}

func (receiver *DataReadWriter) ReadDatagram(controller Controller, from string, data []byte) error {

	if receiver.format != packets.PacketFormatNBOrigin && receiver.format != packets.PacketFormatNB {
		utils.LogError("封包解包器未能就绪, 连接 %s 将会被强行关闭", controller.GetSource())
		return errors.ErrorPacketFormatNotReady
	}

	if receiver.codec != codecs.CodecIMv1 && receiver.codec != codecs.CodecIMv2 {
		utils.LogError("编解码器未能就绪, 连接 %s 将会被强行关闭", controller.GetSource())
		return errors.ErrorCodecNotReady
	}

	packetData := data

	if receiver.format.UnixNeed {
		err, packet, _ := receiver.format.Parser.Pop(data)
		if err != nil {
			utils.LogError("!!! 封包解包失败，连接 %s 将被关闭", controller.GetSource())
			return err
		}

		if packet == nil {
			utils.LogError("!!! 封包解包失败，连接 %s 将被关闭", controller.GetSource())
			return errors.ErrorDataNotMatch
		}

		packetData = packet.Raw
	}

dataDecode:
	//开始使用解码器进行消息解码(单个封包允许包含多个消息体，所以此处有label供goto回流继续解码下一块消息体)
	err, msg, remianData := receiver.codec.Decoder.Decode(packetData)
	if err == nil {
		if receiver.OnDataDecoded != nil {
			err := receiver.OnDataDecoded(controller, controller.GetSource(), msg)
			if err != nil {
				utils.LogError("逻辑处理返回错误 > %s, 连接 %s 将会被强行关闭", err.Error(), controller.GetSource())
				return err
			}
		}
		packetData = remianData
		if len(packetData) > 0 {
			goto dataDecode
		}
	} else {
		utils.LogWarn("进行数据解码失败, 连接 %s 将会被强行关闭", controller.GetSource())
		return err
	}
	return nil
}

func (receiver *DataReadWriter) PackDatagram(controller Controller, msgs ...codecs.IMData) ([]byte, []codecs.IMData, error) {

	if controller != nil && receiver.format != packets.PacketFormatNBOrigin && receiver.format != packets.PacketFormatNB {
		utils.LogError("封包打包器未能就绪, 连接 %s 将会被强行关闭", controller.GetSessionID())
		return []byte(""), msgs, errors.ErrorPacketFormatNotReady
	}

	if controller != nil && receiver.codec != codecs.CodecIMv1 && receiver.codec != codecs.CodecIMv2 {
		utils.LogError("编解码器未能就绪, 连接 %s 将会被强行关闭", controller.GetSessionID())
		return []byte(""), msgs, errors.ErrorCodecNotReady
	}

	var errorMsgs []codecs.IMData = nil
	encodeDatas := make([][]byte, 0)

	for i, msg := range msgs {
		err, data := receiver.codec.Encoder.Encode(&msg)
		if err == nil {
			encodeDatas = append(encodeDatas, data)
		} else {
			if controller != nil {
				utils.LogError("连接 %s 编解码器返回了一个错误", controller.GetSessionID(), err)
			}
			errorMsgs = msgs[i:]
			break
		}
	}

	finalData := bytes.Join(encodeDatas, []byte(""))
	data := finalData

	if receiver.format.UnixNeed {
		packet := packets.Packet{
			Encrypted:       false,
			Compressed:      false,
			CompressSupport: false,
		}
		packet.ProtocolType = receiver.codec.Protocol
		packet.ProtocolVer = receiver.codec.Version

		err, sdata := receiver.format.Packager.Package(&packet, finalData)
		if err != nil {
			return []byte(""), msgs, err
		}
		data = sdata
	}

	return data, errorMsgs, nil
}
