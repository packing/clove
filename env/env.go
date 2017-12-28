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

package env

import (
	"nbpy/codecs"
	"nbpy/packets"
	"nbpy/errors"
	"reflect"
	"bytes"
	"sort"
)

var collectionCodecs map[uint32] *codecs.Codec
var collectionPacketFormats []*packets.PacketFormat


func RegisterCodec(codec *codecs.Codec) (error) {
	k := uint32(codec.Protocol) << 16 | uint32(codec.Version)
	if collectionCodecs == nil {
		collectionCodecs = make(map[uint32] *codecs.Codec)
	}

	_, ok := collectionCodecs[k]
	if ok {
		return errors.Errorf("The codec protocol %d and version %d is exists.", codec.Protocol, codec.Version)
	}

	collectionCodecs[k] = codec
	return nil
}

func FindCodec(protocol uint16, version uint16) (error, *codecs.Codec) {
	k := uint32(protocol) << 16 | uint32(version)
	if collectionCodecs == nil {
		collectionCodecs = make(map[uint32] *codecs.Codec)
	}
	t, ok := collectionCodecs[k]
	if ok {
		return nil, t
	}
	return errors.Errorf("Codec protocol %d and version %d is not exists.", protocol, version), nil
}

func RegisterPacketFormat(fmt *packets.PacketFormat) (error) {
	if collectionPacketFormats == nil {
		collectionPacketFormats = make([]*packets.PacketFormat, 0)
	}

	ok := false
	for _, f := range collectionPacketFormats {
		if f == fmt {
			ok = true
			break
		}
	}
	if ok {
		return errors.Errorf("The packetformat %s is exists.", reflect.TypeOf(*fmt).Kind())
	}

	collectionPacketFormats = append(collectionPacketFormats, fmt)
	return nil
}

func MatchPacketFormat(data *bytes.Buffer) (error, *packets.PacketFormat) {
	formats := collectionPacketFormats
	sort.Slice(formats, func (i, j int) bool{
		return formats[i].Priority > formats[j].Priority
	})
	for _, v := range formats {
		err, b := v.Parser.TryParse(data)
		if err == packets.ErrorDataNotReady {
			return err, nil
		}
		if b {
			return nil, v
		}
	}
	return packets.ErrorDataNotMatch, nil
}