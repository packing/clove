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
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"strconv"
	"unsafe"

	nberrors "github.com/packing/clove/errors"
	nbutils "github.com/packing/clove/utils"
)

const (
	IMDataHeaderLength = 1 + 4
	IMDataTypeUnknown  = 0
	IMDataTypeInt      = 1
	IMDataTypeInt64    = 2
	IMDataTypeStr      = 3
	IMDataTypeFloat    = 4
	IMDataTypeBool     = 5
	IMDataTypeDouble   = 6
	IMDataTypeMap      = 7
	IMDataTypeList     = 8
	IMDataTypeBuffer   = 9
	IMDataTypePBuffer  = 10
	IMDataTypeMemory   = 11
	IMDataTypePyStr    = 12
	IMDataTypeLong     = 13
	IMDataTypeShort    = 14
	IMDataTypeChar     = 15
)

/*

 Intermediate V1 struct
    |header|data|

    header(5 byte) =>>
        |type|size|

        type: 1 byte
        size: 4 byte

    data =>>
        if data's type is map or list, size is elements count of data
        otherwise, size is memory length of data

*/
type DecoderIMv1 struct {
	byteOrder binary.ByteOrder
}

type EncoderIMv1 struct {
	byteOrder binary.ByteOrder
}

func (receiver *DecoderIMv1) SetByteOrder(byteOrder binary.ByteOrder) {
	receiver.byteOrder = byteOrder
}

func (receiver DecoderIMv1) getByteOrder() binary.ByteOrder {
	if receiver.byteOrder == nil {
		return binary.LittleEndian
	}
	return receiver.byteOrder
}

func (receiver DecoderIMv1) Decode(raw []byte) (error, IMData, []byte) {
	if len(raw) == 0 {
		return nberrors.ErrorDataNotEnough, nil, raw
	}
	if len(raw) < IMDataHeaderLength {
		return nberrors.ErrorDataTooShort, nil, raw
	}

	dataType := int8(raw[0])
	count := receiver.getByteOrder().Uint32(raw[1:5])
	switch dataType {
	case IMDataTypeMap:
		return receiver.readMap(raw)
	case IMDataTypeList:
		return receiver.readSlice(raw)
	default:
		return receiver.readMemoryData(raw[IMDataHeaderLength:], dataType, count)
	}

	return nil, nil, raw
}

func (receiver DecoderIMv1) readMemoryData(data []byte, dt int8, size uint32) (error, IMData, []byte) {
	if uint(len(data)) < uint(size) {
		//utils.LogInfo(" %d <-> %d", len(data), size)
		return nberrors.ErrorDataTooShort, nil, data
	}
	switch dt {
	case IMDataTypeInt:
		if size != 4 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, int(receiver.getByteOrder().Uint32(data[:size])), data[size:]
	case IMDataTypeInt64:
		if size != 8 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		if unsafe.Sizeof(0x0) == 8 {
			return nil, int(receiver.getByteOrder().Uint64(data[:size])), data[size:]
		} else {
			return nil, int64(receiver.getByteOrder().Uint64(data[:size])), data[size:]
		}
	case IMDataTypeStr:
		fallthrough
	case IMDataTypePyStr:
		if size > 4096 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, string(data[:size]), data[size:]
	case IMDataTypeFloat:
		if size != 4 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, math.Float32frombits(receiver.getByteOrder().Uint32(data[:size])), data[size:]
	case IMDataTypeBool:
		if size != 1 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, data[0] != 0, data[size:]
	case IMDataTypeDouble:
		if size != 8 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, math.Float64frombits(receiver.getByteOrder().Uint64(data[:size])), data[size:]
	case IMDataTypeMap:
		return receiver.readMap(data)
	case IMDataTypeList:
		return receiver.readSlice(data)
	case IMDataTypeLong:
		if size == 8 {
			if unsafe.Sizeof(0x0) == 8 {
				return nil, int(receiver.getByteOrder().Uint64(data[:size])), data[size:]
			} else {
				return nil, int64(receiver.getByteOrder().Uint64(data[:size])), data[size:]
			}
		} else if size == 4 {
			return nil, int(receiver.getByteOrder().Uint32(data[:size])), data[size:]
		} else {
			return nberrors.ErrorDataIsDamage, nil, data
		}
	case IMDataTypeShort:
		if size != 2 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, int(receiver.getByteOrder().Uint16(data[:size])), data[size:]
	case IMDataTypeChar:
		if size != 1 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, int(data[0]), data[size:]

	case IMDataTypeBuffer:
		fallthrough
	case IMDataTypePBuffer:
		fallthrough
	case IMDataTypeMemory:
		if size > 4096 {
			return nberrors.ErrorDataIsDamage, nil, data
		}
		return nil, data, data[size:]
	default:
		return nberrors.Errorf("Type %b is not supported", dt), nil, data
	}
	return nil, nil, data
}

func (receiver DecoderIMv1) readMap(data []byte) (error, IMMap, []byte) {
	count := int(receiver.getByteOrder().Uint32(data[1:5]))
	ret := make(IMMap, count)
	remain := data[5:]
	var i = 0
	for ; i < count; i++ {
		itemData := remain
		if len(itemData) < IMDataHeaderLength*2 {
			return nberrors.ErrorDataIsDamage, nil, nil
		}
		keyDataType := int8(itemData[0])
		keyDataSize := receiver.getByteOrder().Uint32(itemData[1:5])
		valueDataType := int8(itemData[5])
		valueDataSize := receiver.getByteOrder().Uint32(itemData[6:10])

		if len(itemData) < int(IMDataHeaderLength*2+keyDataSize+valueDataSize) {
			return nberrors.ErrorDataIsDamage, nil, nil
		}

		err, key, _ := receiver.readMemoryData(itemData[10:], keyDataType, keyDataSize)
		if err != nil {
			return err, nil, remain
		}

		err, val, newRemain := receiver.readMemoryData(itemData[10+keyDataSize:], valueDataType, valueDataSize)
		if err != nil {
			return err, nil, remain
		}

		ret[key] = val
		remain = newRemain
	}
	return nil, ret, remain
}

func (receiver DecoderIMv1) readSlice(data []byte) (error, IMSlice, []byte) {
	count := int(receiver.getByteOrder().Uint32(data[1:5]))
	ret := make(IMSlice, count)
	remain := data[5:]
	var i = 0
	for ; i < count; i++ {

		itemData := remain
		if len(itemData) < IMDataHeaderLength {
			return nberrors.ErrorDataTooShort, ret, remain
		}
		valueDataType := int8(itemData[0])
		valueDataSize := receiver.getByteOrder().Uint32(itemData[1:5])

		if uint(len(itemData)) < uint(IMDataHeaderLength+valueDataSize) {
			return nberrors.ErrorDataTooShort, ret, remain
		}

		err, val, newRemain := receiver.readMemoryData(itemData[5:], valueDataType, valueDataSize)
		if err != nil {
			return err, nil, remain
		}

		ret[i] = val
		remain = newRemain
	}
	return nil, ret, remain
}

func (receiver *EncoderIMv1) SetByteOrder(byteOrder binary.ByteOrder) {
	receiver.byteOrder = byteOrder
}

func (receiver EncoderIMv1) getByteOrder() binary.ByteOrder {
	if receiver.byteOrder == nil {
		return binary.LittleEndian
	}
	return receiver.byteOrder
}

func (receiver EncoderIMv1) encodeValueHeader(data *IMData, datasize uint32) (error, []byte) {
	var size uint32 = 0
	tp := IMDataTypeUnknown
	kind := reflect.ValueOf(*data).Type().Kind()
	switch kind {
	case reflect.Int:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Uint32:
		size = 4
		tp = IMDataTypeInt
	case reflect.Int64:
		fallthrough
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
			size = datasize
			if size == 0 {
				size = uint32(len(tmap))
			} else if size == 0 {
			}
			tp = IMDataTypeMap
		} else {
			tlist, oklist := (*data).(IMSlice)
			if oklist {
				size = datasize
				if size == 0 {
					size = uint32(len(tlist))
				} else if size == 0 {
				}
				tp = IMDataTypeList
			} else {
				return nberrors.Errorf("Type %s is not supported", reflect.ValueOf(data).Type().Kind()), nil
			}
		}
	}
	b := make([]byte, IMDataHeaderLength)
	b[0] = byte(tp)
	receiver.getByteOrder().PutUint32(b[1:IMDataHeaderLength], size)
	return nil, b
}

func (receiver EncoderIMv1) encodeValueWithoutHeader(data *IMData) (error, []byte) {
	var b []byte
	switch reflect.ValueOf(*data).Type().Kind() {
	case reflect.Int:
		b = make([]byte, 4)
		receiver.getByteOrder().PutUint32(b, uint32((*data).(int)))
	case reflect.Uint:
		b = make([]byte, 4)
		receiver.getByteOrder().PutUint32(b, uint32((*data).(uint)))
	case reflect.Int32:
		b = make([]byte, 4)
		receiver.getByteOrder().PutUint32(b, uint32((*data).(int32)))
	case reflect.Uint32:
		b = make([]byte, 4)
		receiver.getByteOrder().PutUint32(b, (*data).(uint32))
	case reflect.Int64:
		b = make([]byte, 8)
		receiver.getByteOrder().PutUint64(b, uint64((*data).(int64)))
	case reflect.Uint64:
		b = make([]byte, 8)
		receiver.getByteOrder().PutUint64(b, (*data).(uint64))
	case reflect.Float32:
		b = make([]byte, 4)
		receiver.getByteOrder().PutUint32(b, math.Float32bits((*data).(float32)))
	case reflect.Float64:
		b = make([]byte, 8)
		receiver.getByteOrder().PutUint64(b, math.Float64bits((*data).(float64)))
	case reflect.Bool:
		b = make([]byte, 1)
		b[0] = 0
		if (*data).(bool) {
			b[0] = 1
		}
	case reflect.String:
		str := (*data).(string)
		b = []byte(str)
	case reflect.Int8:
		b = make([]byte, 1)
		b[0] = byte((*data).(int8))
	case reflect.Int16:
		b = make([]byte, 2)
		receiver.getByteOrder().PutUint16(b, uint16((*data).(int16)))
	default:
		tmap, okmap := (*data).(IMMap)
		if okmap {
			var buff bytes.Buffer
			for k, v := range tmap {
				err, kb := receiver.encodeItemValue(&k)
				if err != nil {
					continue
				}
				err, vb := receiver.encodeItemValue(&v)
				if err != nil {
					continue
				}
				err, khb := receiver.encodeValueHeader(&k, 0)
				if err != nil {
					continue
				} else {
				}
				err, vhb := receiver.encodeValueHeader(&v, uint32(len(vb)))
				if err != nil {
					continue
				}
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
					err, vb := receiver.encodeItemValue(&v)
					if err != nil {
						continue
					}
					err, vh := receiver.encodeValueHeader(&v, uint32(len(vb)))
					if err != nil {
						continue
					}
					buff.Write(vh)
					buff.Write(vb)
				}
				b = buff.Bytes()
			} else {
				return nberrors.Errorf("Type %s is not supported", reflect.ValueOf(data).Type().String()), nil
			}
		}
	}

	return nil, b
}

func (receiver EncoderIMv1) encodeItemValue(data *IMData) (error, []byte) {
	_, okmap := (*data).(IMMap)
	_, oklist := (*data).(IMSlice)
	if okmap || oklist {
		var b = make([][]byte, 2)
		var errh error
		errh, b[1] = receiver.encodeValueWithoutHeader(data)
		if errh != nil {
			return errh, nil
		}
		errh, b[0] = receiver.encodeValueHeader(data, 0)
		if errh != nil {
			return errh, nil
		}
		return nil, bytes.Join(b, []byte(""))
	} else {
		return receiver.encodeValueWithoutHeader(data)
	}
}

func (receiver EncoderIMv1) Encode(raw *IMData) (error, []byte) {
	var b = make([][]byte, 2)
	var errh error
	errh, b[1] = receiver.encodeValueWithoutHeader(raw)
	if errh != nil {
		return errh, nil
	}
	errh, b[0] = receiver.encodeValueHeader(raw, 0)
	if errh != nil {
		return errh, nil
	}
	return nil, bytes.Join(b, []byte(""))
}

/*

 Intermediate V2 struct
    [header]|data|

	header 编码 >>
		如果是整数数值类型而且 <= 0x7f, 没有头，第一个字节就是该元素的值, 也就是没有header, 默认数据类型为 char / int8
		如果不是整数数值类型或者数值 > 0x7f, 第一个字节为头信息，标明了该元素的类型，其结构将如下

			| | | | | | | |
			7 6 5 4 3 2 1 0

			其中，最高位 7 一定为1, 作为头信息标识位
			位 5-6 指明本元素数据长度所在字节范围, 有如下取值:
				[ 0 0 ](0): 表明本元素无须表明数据长度(所有数值类型)
				[ 0 1 ](1): 表明本元素数据长度将使用后续1个字节存放(长度为 1 ~ 255，注意所有长度取值将以无符号整数为基准)
				[ 1 0 ](2): 表明本元素数据长度将使用后续2个字节存放(长度为 256 ~ 65,535)
				[ 1 1 ](3): 表明本元素数据长度将使用后续4个字节存放(长度为 65,536 ~ 4,294,967,295)

			位 0-4 用来标识元素的数据类型, 当前版本有如下取值:
				[ 0 0 0 0 0 ](0): 	未知数据 						(保留项，将不会被实际编码, 解码时一定不允许出现)
				[ 0 0 0 0 1 ](1): 	char / int8 					(有符号1字节整数)
				[ 0 0 0 1 0 ](2): 	unsigned char / uint8 			(无符号1字节整数)
				[ 0 0 0 1 1 ](3): 	short / int16 					(有符号2字节整数)
				[ 0 0 1 0 0 ](4): 	unsigned short / uint16 		(无符号2字节整数)
				[ 0 0 1 0 1 ](5): 	int / int32 					(有符号4字节整数)
				[ 0 0 1 1 0 ](6): 	unsigned int / uint32 			(无符号4字节整数)
				[ 0 0 1 1 1 ](7): 	int64 / int64 					(有符号8字节整数)
				[ 0 1 0 0 0 ](8): 	unsigned int64 / uint64 		(无符号8字节整数)
				[ 0 1 0 0 1 ](9): 	float / float32 				(4字节浮点数)
				[ 0 1 0 1 0 ](10): 	double / float64 				(8字节浮点数)
				[ 0 1 0 1 1 ](11): 	reserved 						(无心的保留值, fk)
				[ 0 1 1 0 0 ](12): 	bool / bool true 				(布尔值真 true)
				[ 0 1 1 0 1 ](13): 	bool / bool false				(布尔值否 false)
				[ 0 1 1 1 0 ](14): 	string / string					(字符串，!此类型需要标明长度)
				[ 0 1 1 1 1 ](15): 	char[] / []byte					(二进制数据，!此类型需要标明长度)
				[ 1 0 0 0 0 ](16): 	map / map						(字典类型数据，!此类型需要在长度字节标明元素个数)
				[ 1 0 0 0 1 ](17): 	list / slice					(列表类型数据，!此类型需要在长度字节标明元素个数)

*/

const (
	IMV2DataTypeUnknown     = 0
	IMV2DataTypeInt8        = 1
	IMV2DataTypeUint8       = 2
	IMV2DataTypeInt16       = 3
	IMV2DataTypeUint16      = 4
	IMV2DataTypeInt32       = 5
	IMV2DataTypeUint32      = 6
	IMV2DataTypeInt64       = 7
	IMV2DataTypeUint64      = 8
	IMV2DataTypeFloat32     = 9
	IMV2DataTypeFloat64     = 10
	IMV2DataTypeReserved    = 11
	IMV2DataTypeTrue        = 12
	IMV2DataTypeFalse       = 13
	IMV2DataTypeString      = 14
	IMV2DataTypeBytes       = 15
	IMV2DataTypeMap         = 16
	IMV2DataTypeList        = 17
	IMV2DataTypeJSBigNumber = 18
)

type DecoderIMv2 struct {
	byteOrder binary.ByteOrder
}

type EncoderIMv2 struct {
	byteOrder binary.ByteOrder
}

func calculateTypeSize(o interface{}) uint32 {
	return uint32(reflect.TypeOf(o).Len())
}

func calculateIMV2TypeSize(imtp byte) uint32 {
	switch imtp {
	case IMV2DataTypeInt8:
		fallthrough
	case IMV2DataTypeUint8:
		return 1
	case IMV2DataTypeInt16:
		fallthrough
	case IMV2DataTypeUint16:
		return 2
	case IMV2DataTypeFloat32:
		fallthrough
	case IMV2DataTypeInt32:
		fallthrough
	case IMV2DataTypeUint32:
		return 4
	case IMV2DataTypeFloat64:
		fallthrough
	case IMV2DataTypeInt64:
		fallthrough
	case IMV2DataTypeUint64:
		return 8
	default:
		return 0
	}
}

func makeHeader(tp byte, lenSize byte) byte {
	var h byte = 0x80
	h |= tp & 0x1f
	switch lenSize {
	case 1:
		h |= 0x20
	case 2:
		h |= 0x40
	case 4:
		h |= 0x60
	}
	return h
}

func (receiver EncoderIMv2) makeHeaderAndLength(tp byte, lenData int) []byte {
	var b = make([]byte, 5)
	var lenSize byte = 4
	var lenb = lenData
	switch {
	case lenb <= 0xff:
		lenSize = 1
		b[1] = byte(lenb)
	case lenb <= 0xffff:
		lenSize = 2
		receiver.getByteOrder().PutUint16(b[1:], uint16(lenb))
	default:
		lenSize = 4
		receiver.getByteOrder().PutUint32(b[1:], uint32(lenb))
	}
	b[0] = makeHeader(tp, lenSize)
	b = b[:1+lenSize]
	return b
}

func (receiver *DecoderIMv2) SetByteOrder(byteOrder binary.ByteOrder) {
	receiver.byteOrder = byteOrder
}

func (receiver DecoderIMv2) getByteOrder() binary.ByteOrder {
	if receiver.byteOrder == nil {
		return binary.BigEndian
	}
	return receiver.byteOrder
}

func (receiver DecoderIMv2) Decode(raw []byte) (error, IMData, []byte) {
	defer func() {
		nbutils.LogPanic(recover())
	}()
	if len(raw) == 0 {
		return nberrors.ErrorDataNotEnough, nil, raw
	}
	if len(raw) < 1 {
		return nberrors.ErrorDataTooShort, nil, raw
	}

	fByte := raw[0]
	opCode := fByte >> 7
	if opCode == 0 {
		return nil, int(fByte), raw[1:]
	}

	elementType := fByte & 0x1f
	realData := raw
	lenSizeType := (fByte >> 5) & 0x3
	var elementCount uint32 = 0
	switch lenSizeType {
	case 1:
		if len(raw) < 2 {
			return nberrors.ErrorDataTooShort, nil, raw
		}
		realData = raw[2:]
		elementCount = uint32(raw[1])
	case 2:
		if len(raw) < 3 {
			return nberrors.ErrorDataTooShort, nil, raw
		}
		realData = raw[3:]
		elementCount = uint32(receiver.getByteOrder().Uint16(raw[1:3]))
	case 3:
		if len(raw) < 5 {
			return nberrors.ErrorDataTooShort, nil, raw
		}
		realData = raw[5:]
		elementCount = receiver.getByteOrder().Uint32(raw[1:5])
	default:
		realData = raw[1:]
		elementCount = calculateIMV2TypeSize(elementType)
	}

	if elementType != IMV2DataTypeMap && elementType != IMV2DataTypeList {
		if uint32(len(realData)) < elementCount {
			//如果数据长度不符合预期，可以断定非法数据
			return nberrors.ErrorDataTooShort, nil, nil
		}
	}

	switch elementType {
	case IMV2DataTypeReserved:
		return nil, nil, realData[elementCount:]
	case IMV2DataTypeInt8:
		return nil, int(int8(realData[0])), realData[elementCount:]
	case IMV2DataTypeUint8:
		return nil, uint(realData[0]), realData[elementCount:]
	case IMV2DataTypeInt16:
		return nil, int(int16(receiver.getByteOrder().Uint16(realData))), realData[elementCount:]
	case IMV2DataTypeUint16:
		return nil, uint(receiver.getByteOrder().Uint16(realData)), realData[elementCount:]
	case IMV2DataTypeInt32:
		return nil, int(int32(receiver.getByteOrder().Uint32(realData))), realData[elementCount:]
	case IMV2DataTypeUint32:
		return nil, uint(receiver.getByteOrder().Uint32(realData)), realData[elementCount:]
	case IMV2DataTypeInt64:
		if unsafe.Sizeof(0x0) == 8 {
			return nil, int(int64(receiver.getByteOrder().Uint64(realData))), realData[elementCount:]
		} else {
			return nil, int64(receiver.getByteOrder().Uint64(realData)), realData[elementCount:]
		}
	case IMV2DataTypeUint64:
		if unsafe.Sizeof(0x0) == 8 {
			return nil, uint(receiver.getByteOrder().Uint64(realData)), realData[elementCount:]
		} else {
			return nil, receiver.getByteOrder().Uint64(realData), realData[elementCount:]
		}
	case IMV2DataTypeFloat32:
		return nil, math.Float32frombits(receiver.getByteOrder().Uint32(realData)), realData[elementCount:]
	case IMV2DataTypeFloat64:
		return nil, math.Float64frombits(receiver.getByteOrder().Uint64(realData)), realData[elementCount:]
	case IMV2DataTypeTrue:
		return nil, true, realData
	case IMV2DataTypeFalse:
		return nil, false, realData
	case IMV2DataTypeString:
		e := string(realData[:elementCount])
		return nil, e, realData[elementCount:]
	case IMV2DataTypeJSBigNumber:
		e := string(realData[:elementCount])
		var rr interface{} = 0
		er, errpar := strconv.ParseInt(e, 10, 64)
		if errpar != nil {
			return nil, 0, realData[elementCount:]
		}
		if unsafe.Sizeof(0x0) == 8 {
			rr = int(er)
		} else {
			rr = er
		}
		return nil, rr, realData[elementCount:]
	case IMV2DataTypeBytes:
		return nil, realData[:elementCount], realData[elementCount:]
	case IMV2DataTypeMap:
		var dstMap = make(IMMap, elementCount)
		for i := 0; uint32(i) < elementCount; i++ {
			err, kd, remain := receiver.Decode(realData)
			if err != nil {
				return err, nil, nil
			}
			realData = remain
			err, vd, remain := receiver.Decode(realData)
			if err != nil {
				return err, nil, nil
			}
			realData = remain
			dstMap[kd] = vd
		}
		return nil, dstMap, realData
	case IMV2DataTypeList:
		var dstList = make(IMSlice, 0)
		for i := 0; uint32(i) < elementCount; i++ {
			err, vd, remain := receiver.Decode(realData)
			if err != nil {
				return err, nil, nil
			}
			realData = remain
			dstList = append(dstList, vd)
		}
		return nil, dstList, realData
	}
	return nil, nil, raw
}

func (receiver *EncoderIMv2) SetByteOrder(byteOrder binary.ByteOrder) {
	receiver.byteOrder = byteOrder
}

func (receiver EncoderIMv2) getByteOrder() binary.ByteOrder {
	if receiver.byteOrder == nil {
		return binary.BigEndian
	}
	return receiver.byteOrder
}

func (receiver EncoderIMv2) Encode(raw *IMData) (error, []byte) {
	defer func() {
		nbutils.LogPanic(recover())
	}()
	if *raw == nil {
		rb := make([]byte, 1)
		rb[0] = makeHeader(IMV2DataTypeReserved, 0)
		return nil, rb
		//return nberrors.ErrorTypeNotSupported, []byte("")
	}
	var rawValue = reflect.ValueOf(*raw)
	var tpKind = rawValue.Type().Kind()

	var isInt = false
	var isSign = false
	var isFloat = false
	var isMap = false
	var isSlice = false
	var intRawValue int64 = 0
	var intValue uint64 = 0
	switch tpKind {
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Int:
		intRawValue = reflect.ValueOf(*raw).Int()

		intValue = uint64(intRawValue)
		//utils.LogInfo("sign raw: %d, int %d", intRawValue, intValue)
		isInt = true
		isSign = intRawValue < 0

	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Uint:
		intValue = uint64(reflect.ValueOf(*raw).Uint())
		isInt = true
		isSign = false

	case reflect.Float32:
		isFloat = true
		isSign = false
		intValue = uint64(math.Float32bits((*raw).(float32)))
	case reflect.Float64:
		isFloat = true
		isSign = false
		intValue = math.Float64bits((*raw).(float64))
	case reflect.Bool:
		rb := make([]byte, 1)
		if reflect.ValueOf(*raw).Bool() {
			rb[0] = makeHeader(IMV2DataTypeTrue, 0)
		} else {
			rb[0] = makeHeader(IMV2DataTypeFalse, 0)
		}
		return nil, rb
	case reflect.Map:
		isMap = true
	case reflect.Slice:
		isSlice = true
	default:
	}

	if isInt {
		//进行压缩处理
		rb := make([]byte, 9)

		if isSign {
			switch {
			case intRawValue <= 127 && intRawValue >= -128:
				rb[0] = makeHeader(IMV2DataTypeInt8, 0)
				rb[1] = byte(intRawValue)
				return nil, rb[:2]
			case intRawValue <= -129 && intRawValue >= -32768:
				rb[0] = makeHeader(IMV2DataTypeInt16, 0)
				receiver.getByteOrder().PutUint16(rb[1:], uint16(int16(intRawValue)))
				return nil, rb[:3]
			case intRawValue < -32768 && intRawValue >= -2147483648:
				rb[0] = makeHeader(IMV2DataTypeInt32, 0)
				receiver.getByteOrder().PutUint32(rb[1:], uint32(int32(intRawValue)))
				return nil, rb[:5]
			default:
				rb[0] = makeHeader(IMV2DataTypeInt64, 0)
				receiver.getByteOrder().PutUint64(rb[1:], uint64(intRawValue))
				return nil, rb
			}
		} else {
			switch {
			case intValue <= 0x7f:
				rb[0] = byte(intValue)
				return nil, rb[:1]
			case intValue <= 0xff:
				rb[0] = makeHeader(IMV2DataTypeUint8, 0)
				rb[1] = byte(intValue)
				return nil, rb[:2]
			case intValue <= 0xffff:
				rb[0] = makeHeader(IMV2DataTypeUint16, 0)
				receiver.getByteOrder().PutUint16(rb[1:], uint16(intValue))
				return nil, rb[:3]
			case intValue <= 0xffffffff:
				rb[0] = makeHeader(IMV2DataTypeUint32, 0)
				receiver.getByteOrder().PutUint32(rb[1:], uint32(intValue))
				return nil, rb[:5]
			default:
				rb[0] = makeHeader(IMV2DataTypeUint64, 0)
				receiver.getByteOrder().PutUint64(rb[1:], intValue)
				return nil, rb
			}
		}
	}

	if isFloat {
		rb := make([]byte, 9)
		if intValue <= 0xffffffff {
			rb[0] = makeHeader(IMV2DataTypeFloat32, 0)
			receiver.getByteOrder().PutUint32(rb[1:], uint32(intValue))
			return nil, rb[:5]
		} else {
			rb[0] = makeHeader(IMV2DataTypeFloat64, 0)
			receiver.getByteOrder().PutUint64(rb[1:], uint64(intValue))
			return nil, rb[:9]
		}
	}

	bytesRaw, isBytes := (*raw).([]byte)
	strRaw, isStr := (*raw).(string)
	if isBytes || isStr {
		rbs := make([][]byte, 2)
		rbs[1] = bytesRaw
		var etp byte = IMV2DataTypeBytes
		if isStr {
			etp = IMV2DataTypeString
			rbs[1] = []byte(strRaw)
		}
		rbs[0] = receiver.makeHeaderAndLength(etp, len(rbs[1]))

		return nil, bytes.Join(rbs, []byte(""))
	}

	//mapRaw, isMap := (*raw).(IMMap)
	isMap = tpKind == reflect.Map
	if isMap {
		var i = 0
		var rbs = make([][]byte, rawValue.Len()+1)
		var ks = rawValue.MapKeys()
		for ki := range ks {
			var k = ks[ki]
			var v = rawValue.MapIndex(k)
			var ik = k.Interface()
			err, rk := receiver.Encode(&ik)
			if err != nil {
				continue
			}
			var iv = v.Interface()
			err, vk := receiver.Encode(&iv)
			if err != nil {
				continue
			}
			rbs[i+1] = bytes.Join([][]byte{rk, vk}, []byte(""))
			i++
		}

		rbs[0] = receiver.makeHeaderAndLength(IMV2DataTypeMap, i)

		return nil, bytes.Join(rbs, []byte(""))
	}

	isSlice = tpKind == reflect.Slice
	if isSlice {
		var totalLen = rawValue.Len()
		var rbs = make([][]byte, totalLen+1)
		for i := 0; i < totalLen; i++ {
			v := rawValue.Index(i)
			iv := v.Interface()
			err, vk := receiver.Encode(&iv)
			if err != nil {
				continue
			}
			rbs[i+1] = vk
		}

		rbs[0] = receiver.makeHeaderAndLength(IMV2DataTypeList, totalLen)

		return nil, bytes.Join(rbs, []byte(""))
	}

	return nberrors.ErrorTypeNotSupported, nil
}

type IntermediateValue struct {
	raw interface{}
}

func CreateIntermediateValue(raw interface{}) *IntermediateValue {
	v := new(IntermediateValue)
	v.raw = raw
	return v
}

func (receiver IntermediateValue) String() string {
	s, ok := receiver.raw.(string)
	if !ok {
		return ""
	}
	return s
}

func (receiver IntermediateValue) Int() int {
	return receiver.IntWithDefault(0)
}

func (receiver IntermediateValue) Int64() int64 {
	return receiver.Int64WithDefault(0)
}

func (receiver IntermediateValue) IntWithDefault(def int) int {
	s, ok := receiver.raw.(int)
	if !ok {
		return def
	}
	return s
}

func (receiver IntermediateValue) Int64WithDefault(def int64) int64 {
	s, ok := receiver.raw.(int64)
	if !ok {
		return def
	}
	return s
}

func (receiver IntermediateValue) Bool() bool {
	s, ok := receiver.raw.(bool)
	if !ok {
		return false
	}
	return s
}

func (receiver IntermediateValue) Float32() float32 {
	s, ok := receiver.raw.(float32)
	if !ok {
		return 0
	}
	return s
}

func (receiver IntermediateValue) Float64() float64 {
	s, ok := receiver.raw.(float64)
	if !ok {
		return 0
	}
	return s
}

func (receiver IntermediateValue) Map() map[interface{}]interface{} {
	s, ok := receiver.raw.(map[interface{}]interface{})
	if !ok {
		return make(map[interface{}]interface{})
	}
	return s
}

func (receiver IntermediateValue) Slice() []interface{} {
	s, ok := receiver.raw.([]interface{})
	if !ok {
		return make([]interface{}, 0)
	}
	return s
}

var codecIMv1 = Codec{Protocol: ProtocolIM, Version: 1, Decoder: new(DecoderIMv1), Encoder: new(EncoderIMv1), Name: "结构化中间数据流v1"}
var CodecIMv1 = &codecIMv1

var codecIMv2 = Codec{Protocol: ProtocolIM, Version: 2, Decoder: new(DecoderIMv2), Encoder: new(EncoderIMv2), Name: "结构化中间数据流v2"}
var CodecIMv2 = &codecIMv2
