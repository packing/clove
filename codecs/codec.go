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
	"../errors"
	"reflect"
)

const (
	ProtocolMemory  	= 0x0
	ProtocolIM 			= 0x1
	ProtocolJSON 		= 0x2
	ProtocolReserved 	= 0xF
)

type IMData = interface{}
type IMMap = map[IMData] IMData
type IMStrMap = map[string] IMData
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
	Name string
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

type IMMapReader struct {
	Map IMMap
}

func CreateMapReader(m IMMap) * IMMapReader {
	mr := new(IMMapReader)
	mr.Map = m
	return mr
}

func (receiver IMMapReader) TryReadValue(key interface{}) interface{} {
	kind := reflect.TypeOf(key).Kind()
	switch kind {
	case reflect.String:
		return receiver.Map[key]

	case reflect.Int: fallthrough
	case reflect.Int8: fallthrough
	case reflect.Int16: fallthrough
	case reflect.Int32: fallthrough
	case reflect.Int64:
		intV := reflect.ValueOf(key).Int()
		v, ok := receiver.Map[int(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int8(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int16(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int32(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int64(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint8(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint16(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint32(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint64(intV)]
		if ok {
			return v
		} else {
			return nil
		}

	case reflect.Uint: fallthrough
	case reflect.Uint8: fallthrough
	case reflect.Uint16: fallthrough
	case reflect.Uint32: fallthrough
	case reflect.Uint64:
		intV := reflect.ValueOf(key).Uint()
		v, ok := receiver.Map[uint(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint8(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint16(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint32(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[uint64(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int(intV)]
		if ok {
			return v
		}
		v, ok = receiver.Map[int8(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int16(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int32(intV)]
		if ok {
			return v
		}

		v, ok = receiver.Map[int64(intV)]
		if ok {
			return v
		} else {
			return nil
		}

	case reflect.Float32: fallthrough
	case reflect.Float64:
		floatV := reflect.ValueOf(key).Float()

		v, ok := receiver.Map[floatV]
		if ok {
			return v
		}

		v, ok = receiver.Map[float32(floatV)]
		if ok {
			return v
		} else {
			return nil
		}

	default:
		return nil
	}
}

func (receiver IMMapReader) IntValueOf(key interface{}, def int64) int64 {
	v := receiver.TryReadValue(key)
	if v == nil {
		return def
	}
	kind := reflect.TypeOf(v).Kind()
	switch kind {
	case reflect.Int: fallthrough
	case reflect.Int8: fallthrough
	case reflect.Int16: fallthrough
	case reflect.Int32: fallthrough
	case reflect.Int64:
		return reflect.ValueOf(v).Int()
	case reflect.Uint: fallthrough
	case reflect.Uint8: fallthrough
	case reflect.Uint16: fallthrough
	case reflect.Uint32: fallthrough
	case reflect.Uint64:
		return int64(reflect.ValueOf(v).Uint())
	default:
		return def
	}
}

func (receiver IMMapReader) StrValueOf(key interface{}, def string) string {
	v := receiver.TryReadValue(key)
	if v == nil {
		return def
	}
	if reflect.TypeOf(v).Kind() != reflect.String {
		return def
	}
	return reflect.ValueOf(v).String()
}

func (receiver IMMapReader) FloatValueOf(key interface{}, def float64) float64 {
	v := receiver.TryReadValue(key)
	if v == nil {
		return def
	}
	if reflect.TypeOf(v).Kind() != reflect.Float64 && reflect.TypeOf(v).Kind() != reflect.Float32 {
		return def
	}
	return reflect.ValueOf(v).Float()
}

func (receiver IMMapReader) BoolValueOf(key interface{}) bool {
	v := receiver.TryReadValue(key)
	if v == nil {
		return false
	}
	if reflect.TypeOf(v).Kind() != reflect.Bool {
		return false
	}
	return reflect.ValueOf(v).Bool()
}

type IMSliceReader struct {
	List IMSlice
}

func CreateSliceReader(m IMSlice) * IMSliceReader {
	mr := new(IMSliceReader)
	mr.List = m
	return mr
}

func (receiver IMSliceReader) IntValueOf(index int, def int64) int64 {
	if index >= len(receiver.List) || index < 0 {
		return def
	}
	v := receiver.List[index]
	if v == nil {
		return def
	}

	kind := reflect.TypeOf(v).Kind()
	switch kind {
	case reflect.Int: fallthrough
	case reflect.Int8: fallthrough
	case reflect.Int16: fallthrough
	case reflect.Int32: fallthrough
	case reflect.Int64:
		return reflect.ValueOf(v).Int()
	case reflect.Uint: fallthrough
	case reflect.Uint8: fallthrough
	case reflect.Uint16: fallthrough
	case reflect.Uint32: fallthrough
	case reflect.Uint64:
		return int64(reflect.ValueOf(v).Uint())
	default:
		return def
	}
}

func (receiver IMSliceReader) StrValueOf(index int, def string) string {
	if index >= len(receiver.List) || index < 0 {
		return def
	}
	v := receiver.List[index]
	if reflect.TypeOf(v).Kind() != reflect.String {
		return def
	}
	return reflect.ValueOf(v).String()
}

func (receiver IMSliceReader) MySQLStrValueOf(index int, def string) string {
	if index >= len(receiver.List) || index < 0 {
		return def
	}
	v := receiver.List[index]
	if v == nil {
		return def
	}
	if reflect.TypeOf(v).Kind() != reflect.Slice {
		return def
	}
	return string(reflect.ValueOf(v).Bytes())
}

func (receiver IMSliceReader) FloatValueOf(index int, def float64) float64 {
	if index >= len(receiver.List) || index < 0 {
		return def
	}
	v := receiver.List[index]
	if reflect.TypeOf(v).Kind() != reflect.Float64 && reflect.TypeOf(v).Kind() != reflect.Float32 {
		return def
	}
	return reflect.ValueOf(v).Float()
}

func (receiver IMSliceReader) BoolValueOf(index int) bool {
	if index >= len(receiver.List) || index < 0 {
		return false
	}
	v := receiver.List[index]
	if reflect.TypeOf(v).Kind() != reflect.Bool {
		return false
	}
	return reflect.ValueOf(v).Bool()
}

var codecMemoryV1 = Codec{Protocol:ProtocolMemory, Version:1, Decoder: DecoderMemory{}, Encoder: EncoderMemory{}, Name: "二进制流"}
var CodecMemoryV1 = &codecMemoryV1

func Int64FromInterface(v interface{}) int64 {
	var rv int64 = 0
	switch v.(type) {
	case int: rv = int64(v.(int))
	case int8: rv = int64(v.(int8))
	case int16: rv = int64(v.(int16))
	case int32: rv = int64(v.(int32))
	case int64: rv = int64(v.(int64))
	case uint: rv = int64(v.(uint))
	case uint8: rv = int64(v.(uint8))
	case uint16: rv = int64(v.(uint16))
	case uint32: rv = int64(v.(uint32))
	case uint64: rv = int64(v.(uint64))
	}
	return rv
}

func IntFromInterface(v interface{}) int {
    var rv = 0
    switch v.(type) {
    case int: rv = v.(int)
    case int8: rv = int(v.(int8))
    case int16: rv = int(v.(int16))
    case int32: rv = int(v.(int32))
    case int64: rv = int(v.(int64))
    case uint: rv = int(v.(uint))
    case uint8: rv = int(v.(uint8))
    case uint16: rv = int(v.(uint16))
    case uint32: rv = int(v.(uint32))
    case uint64: rv = int(v.(uint64))
    }
    return rv
}