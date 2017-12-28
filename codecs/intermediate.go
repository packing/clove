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
	"encoding/binary"
	"nbpy/errors"
	"math"
	"bytes"
	"reflect"
)

const (
	IMDataHeaderLength 	= 1 + 4
	IMDataTypeUnknown 	= 0
	IMDataTypeInt 		= 1
	IMDataTypeInt64 	= 2
	IMDataTypeStr 		= 3
	IMDataTypeFloat 	= 4
	IMDataTypeBool 		= 5
	IMDataTypeDouble 	= 6
	IMDataTypeMap 		= 7
	IMDataTypeList 		= 8
	IMDataTypeBuffer 	= 9
	IMDataTypePBuffer 	= 10
	IMDataTypeMemory 	= 11
	IMDataTypePyStr 	= 12
	IMDataTypeLong 		= 13
	IMDataTypeShort 	= 14
	IMDataTypeChar 		= 15
)

type DecoderIMv1 struct {
}

type EncoderIMv1 struct {
}

func (receiver DecoderIMv1) Decode(raw []byte) (error, IMData, []byte){
	if len(raw) < IMDataHeaderLength {
		return errors.Errorf("The length of the head data is not enough to be decoded"), nil, raw
	}

	dataType := int(raw[0])
	switch dataType {
	case IMDataTypeMap:
		return receiver.readMap(raw)
	case IMDataTypeList:
		return receiver.readSlice(raw)
	default:
		return errors.Errorf("Type %b is not supported", dataType), nil, raw
	}
	return nil, nil, raw
}

func (receiver DecoderIMv1) readMemoryData(data []byte, dt int, size uint32) (error, IMData, []byte)  {
	switch dt {
	case IMDataTypeInt:
		return nil, int(binary.LittleEndian.Uint32(data[:size])), data[size:]
	case IMDataTypeInt64:
		return nil, int64(binary.LittleEndian.Uint64(data[:size])), data[size:]
	case IMDataTypeStr: fallthrough
	case IMDataTypePyStr:
		return nil, string(data[:size]), data[size:]
	case IMDataTypeFloat:
		return nil, math.Float32frombits(binary.LittleEndian.Uint32(data[:size])), data[size:]
	case IMDataTypeBool:
		return nil, data[0] != 0, data[size:]
	case IMDataTypeDouble:
		return nil, math.Float64frombits(binary.LittleEndian.Uint64(data[:size])), data[size:]
	case IMDataTypeMap:
		return receiver.readMap(data)
	case IMDataTypeList:
		return receiver.readSlice(data)
	case IMDataTypeLong:
		if size == 8 {
			return nil, int64(binary.LittleEndian.Uint64(data[:size])), data[size:]
		} else if size == 4 {
			return nil, int(binary.LittleEndian.Uint32(data[:size])), data[size:]
		} else {
			return errors.Errorf("The long data size %d is not valid", size), nil, data
		}
	case IMDataTypeShort:
		return nil, int16(binary.LittleEndian.Uint16(data[:size])), data[size:]
	case IMDataTypeChar:
		return nil, data[0], data[size:]

	case IMDataTypeBuffer: fallthrough
	case IMDataTypePBuffer: fallthrough
	case IMDataTypeMemory:
		return nil, data, data[size:]
	default:
		return errors.Errorf("Type %b is not supported", dt), nil, data
	}
	return nil, nil, data
}

func (receiver DecoderIMv1) readMap(data []byte) (error, IMMap, []byte) {
	count := int(binary.LittleEndian.Uint32(data[1:5]))
	ret := make(IMMap, count)
	remain := data[5:]
	for i := 0; i < count; i ++ {
		itemData := remain
		keyDataType := int(itemData[0])
		keyDataSize := binary.LittleEndian.Uint32(itemData[1:5])
		valueDataType := int(itemData[5])
		valueDataSize := binary.LittleEndian.Uint32(itemData[6:10])

		err, key, _ := receiver.readMemoryData(itemData[10:], keyDataType, keyDataSize)
		if err != nil { return err, nil, remain }
		err, val, newRemain := receiver.readMemoryData(itemData[10 + keyDataSize:], valueDataType, valueDataSize)
		if err != nil { return err, nil, remain }

		ret[key] = val
		remain = newRemain
	}
	return nil, ret, remain
}

func (receiver DecoderIMv1) readSlice(data []byte) (error, IMSlice, []byte) {
	count := int(binary.LittleEndian.Uint32(data[1:5]))
	ret := make(IMSlice, count)
	remain := data[5:]
	for i := 0; i < count; i ++ {
		itemData := remain
		valueDataType := int(itemData[0])
		valueDataSize := binary.LittleEndian.Uint32(itemData[1:5])

		err, val, newRemain := receiver.readMemoryData(itemData[5:], valueDataType, valueDataSize)
		if err != nil { return err, nil, remain }

		ret[i] = val
		remain = newRemain
	}
	return nil, ret, remain
}

func (receiver EncoderIMv1) encodeValueHeader(data *IMData) (error, []byte){
	var size uint32 = 0
	tp := IMDataTypeUnknown
	switch reflect.ValueOf(*data).Type().Kind() {
	case reflect.Int: fallthrough
	case reflect.Uint: fallthrough
	case reflect.Int32: fallthrough
	case reflect.Uint32:
		size = 4
		tp = IMDataTypeInt
	case reflect.Int64: fallthrough
	case reflect.Uint64:
		size = 8
		tp = IMDataTypeInt64
	case reflect.Float32:
		size = 4
		tp = IMDataTypeFloat
	case reflect.Float64:
		size = 8
		tp = IMDataTypeDouble
	case reflect.Bool:
		size = 1
		tp = IMDataTypeBool
	case reflect.String:
		size = uint32(len((*data).(string)))
		tp = IMDataTypeStr
	case reflect.Int8:
		size = 1
		tp = IMDataTypeChar
	case reflect.Int16:
		size = 2
		tp = IMDataTypeShort
	default:
		tmap, okmap := (*data).(IMMap)
		if okmap {
			size = uint32(len(tmap))
			tp = IMDataTypeMap
		} else {
			tlist, oklist := (*data).(IMSlice)
			if oklist {
				size = uint32(len(tlist))
				tp = IMDataTypeList
			} else {
				return errors.Errorf("Type %s is not supported", reflect.ValueOf(data).Type().Kind()), nil
			}
		}
	}
	b := make([]byte, IMDataHeaderLength)
	b[0] = byte(tp)
	binary.LittleEndian.PutUint32(b[1:IMDataHeaderLength], size)
	return nil, b
}

func (receiver EncoderIMv1) encodeValueWithoutHeader(data *IMData) (error, []byte) {
	var b []byte
	switch reflect.ValueOf(*data).Type().Kind() {
	case reflect.Int:
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32((*data).(int)))
	case reflect.Uint:
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32((*data).(uint)))
	case reflect.Int32:
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32((*data).(int32)))
	case reflect.Uint32:
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, (*data).(uint32))
	case reflect.Int64:
		b = make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64((*data).(int64)))
	case reflect.Uint64:
		b = make([]byte, 8)
		binary.LittleEndian.PutUint64(b, (*data).(uint64))
	case reflect.Float32:
		b = make([]byte, 4)
		binary.LittleEndian.PutUint32(b, math.Float32bits((*data).(float32)))
	case reflect.Float64:
		b = make([]byte, 8)
		binary.LittleEndian.PutUint64(b, math.Float64bits((*data).(float64)))
	case reflect.Bool:
		b = make([]byte, 1)
		b[0] = 0
		if (*data).(bool) { b[0] = 1 }
	case reflect.String:
		str := (*data).(string)
		b = []byte(str)
	case reflect.Int8:
		b = make([]byte, 1)
		b[0] = byte((*data).(int8))
	case reflect.Int16:
		b = make([]byte, 2)
		binary.LittleEndian.PutUint16(b, uint16((*data).(int16)))
	default:
		tmap, okmap := (*data).(IMMap)
		if okmap {
			var buff bytes.Buffer
			for k,v := range tmap {
				err, khb := receiver.encodeValueHeader(&k)
				if err != nil { continue }
				err, vhb := receiver.encodeValueHeader(&v)
				if err != nil { continue }
				err, kb := receiver.encodeItemValue(&k)
				if err != nil { continue }
				err, vb := receiver.encodeItemValue(&v)
				if err != nil { continue }
				buff.Write(khb)
				buff.Write(vhb)
				buff.Write(kb)
				buff.Write(vb)
			}
			b = buff.Bytes()
		} else {
			tlist, oklist := (*data).(IMSlice)
			if oklist {
				var buff bytes.Buffer
				for _, v := range tlist {
					err, vh := receiver.encodeValueHeader(&v)
					if err != nil { continue }
					err, vb := receiver.encodeValueWithoutHeader(&v)
					if err != nil { continue }
					buff.Write(vh)
					buff.Write(vb)
				}
				b = buff.Bytes()
			} else {
				return errors.Errorf("Type %s is not supported", reflect.ValueOf(data).Type().String()), nil
			}
		}
	}

	return nil, b
}

func (receiver EncoderIMv1) encodeItemValue(data *IMData) (error, []byte){
	_, okmap := (*data).(IMMap)
	_, oklist := (*data).(IMSlice)
	if okmap || oklist {
		var b = make([][]byte, 2)
		var errh error
		errh, b[0] = receiver.encodeValueHeader(data)
		if errh != nil {
			return errh, nil
		}
		errh, b[1] = receiver.encodeValueWithoutHeader(data)
		if errh != nil {
			return errh, nil
		}
		return nil, bytes.Join(b, []byte(""))
	} else {
		return receiver.encodeValueWithoutHeader(data)
	}
}

func (receiver EncoderIMv1) Encode(raw *IMData) (error, []byte){
	var b = make([][]byte, 2)
	var errh error
	errh, b[0] = receiver.encodeValueHeader(raw)
	if errh != nil {
		return errh, nil
	}
	errh, b[1] = receiver.encodeValueWithoutHeader(raw)
	if errh != nil {
		return errh, nil
	}
	return nil, bytes.Join(b, []byte(""))
}



var codecIMv1 = Codec{Protocol:ProtocolIM, Version:1, Decoder: DecoderIMv1{}, Encoder: EncoderIMv1{}}
var CodecIMv1 = &codecIMv1