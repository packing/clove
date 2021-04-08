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
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	nbcodecs "github.com/packing/clove/codecs"
	nberrors "github.com/packing/clove/errors"
	nbpackets "github.com/packing/clove/packets"
	nbutils "github.com/packing/clove/utils"
)

var collectionCodecs map[byte]*nbcodecs.Codec
var collectionPacketFormats []*nbpackets.PacketFormat

func RegisterCodec(codec *nbcodecs.Codec) error {
	k := byte((byte(codec.Protocol) << 4) | byte(codec.Version))
	if collectionCodecs == nil {
		collectionCodecs = make(map[byte]*nbcodecs.Codec)
	}
	nbutils.LogVerbose(">>> 注册编解码器 封包协议[%d][ver.%d] [%s]", codec.Protocol, codec.Version, codec.Name)
	_, ok := collectionCodecs[k]
	if ok {
		return nberrors.Errorf("!!! 封包协议[%d][%s] 版本[%d] 已经注册，无需重复注册", codec.Protocol, codec.Version, codec.Name)
	}

	collectionCodecs[k] = codec
	return nil
}

func FindCodec(protocol byte, version byte) (error, *nbcodecs.Codec) {
	k := byte((protocol << 4) | version)
	if collectionCodecs == nil {
		collectionCodecs = make(map[byte]*nbcodecs.Codec)
	}
	t, ok := collectionCodecs[k]
	if ok {
		return nil, t
	}
	return nberrors.Errorf("!!! 封包协议[%d] 版本[%d] 不存在", protocol, version), nil
}

func RegisterPacketFormat(fmt *nbpackets.PacketFormat) error {
	if collectionPacketFormats == nil {
		collectionPacketFormats = make([]*nbpackets.PacketFormat, 0)
	}

	ok := false
	for _, f := range collectionPacketFormats {
		if f == fmt {
			ok = true
			break
		}
	}
	if ok {
		return nberrors.Errorf("!!! 指定的封包格式 %s 不存在", reflect.TypeOf(*fmt).Kind())
	}

	collectionPacketFormats = append(collectionPacketFormats, fmt)
	return nil
}

func MatchPacketFormat(data []byte) (error, *nbpackets.PacketFormat) {
	formats := collectionPacketFormats
	sort.Slice(formats, func(i, j int) bool {
		return formats[i].Priority > formats[j].Priority
	})
	for _, v := range formats {
		err, b := v.Parser.TryParse(data)
		if err == nberrors.ErrorDataNotReady {
			return err, nil
		}
		if b {
			return nil, v
		}
	}
	return nberrors.ErrorDataNotMatch, nil
}

func Schedule() os.Signal {

	//定时器实现

	c := make(chan os.Signal, 1)
	signals := []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.Signal(0x11),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.Signal(0x1e),
		syscall.Signal(0x1f),
	}
	signal.Notify(c, signals...)
	s := <-c
	nbutils.LogVerbose(">>> 收到信号 > %s", s.String())
	return s
}

func GetGoroutineId() int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("get goroutine id error", err)
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}
