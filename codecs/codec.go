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
	Protocol uint16
	Version uint16
	Decoder Decoder
	Encoder Encoder
}

var ErrorDataNotEnough = errors.Errorf("The length of the head data is not enough to be decoded")
var ErrorDataTooShort = errors.Errorf("The length of the head data is too short")
var ErrorTypeNotSupported = errors.Errorf("Type is not supported")
