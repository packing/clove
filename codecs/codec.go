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

package codecs

import (
	"nbpy/errors"
)

const (
	ProtocolMemory  	= 0x0
	ProtocolIM 			= 0x1
	ProtocolJSON 		= 0x2
	ProtocolReserved 	= 0xF
)

type IMData = interface{}
type IMMap = map[IMData] IMData
type IMSlice = []IMData

type Decoder interface {
	Decode([]byte) (error, IMData, []byte)
}

type Encoder interface {
	Encode(*IMData) (error, []byte)
}

type Codec struct {
	Protocol byte
	Version byte
	Decoder Decoder
	Encoder Encoder
}

type DecoderMemory struct {}
func (receiver DecoderMemory) Decode(raw []byte) (error, IMData, []byte) {
	return nil, raw, raw[len(raw):]
}

type EncoderMemory struct {}
func (receiver EncoderMemory) Encode(raw *IMData) (error, []byte) {
	data, ok := (*raw).([]byte)
	if !ok {
		return errors.ErrorTypeNotSupported, nil
	}
	return nil, data
}

var codecMemoryV1 = Codec{Protocol:ProtocolMemory, Version:1, Decoder: DecoderMemory{}, Encoder: EncoderMemory{}}
var CodecMemoryV1 = &codecMemoryV1