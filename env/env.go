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
	"sort"
	"nbpy/utils"
	"os"
	"syscall"
	"os/signal"
	"fmt"
)

var collectionCodecs map[byte] *codecs.Codec
var collectionPacketFormats []*packets.PacketFormat

func RegisterCodec(codec *codecs.Codec) (error) {
	k := byte((byte(codec.Protocol) << 4) | byte(codec.Version))
	if collectionCodecs == nil {
		collectionCodecs = make(map[byte] *codecs.Codec)
	}
	utils.LogVerbose("RegisterCodec %d[%d] KEY:%d ", codec.Protocol, codec.Version, k)
	_, ok := collectionCodecs[k]
	if ok {
		return errors.Errorf("The codec protocol %d and version %d is exists.", codec.Protocol, codec.Version)
	}

	collectionCodecs[k] = codec
	return nil
}

func FindCodec(protocol byte, version byte) (error, *codecs.Codec) {
	k := byte((protocol << 4) | version)
	if collectionCodecs == nil {
		collectionCodecs = make(map[byte] *codecs.Codec)
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

func MatchPacketFormat(data []byte) (error, *packets.PacketFormat) {
	formats := collectionPacketFormats
	sort.Slice(formats, func (i, j int) bool{
		return formats[i].Priority > formats[j].Priority
	})
	for _, v := range formats {
		err, b := v.Parser.TryParse(data)
		if err == errors.ErrorDataNotReady {
			return err, nil
		}
		if b {
			return nil, v
		}
	}
	return errors.ErrorDataNotMatch, nil
}

func Schedule() {

	//定时器实现

	c := make(chan os.Signal, 1)
	signals := []os.Signal{
		//os.Interrupt,
		//os.Kill,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGSTOP,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
	}
	signal.Notify(c, signals...)
	s := <-c
	fmt.Println("收到信号 > ", s)
}