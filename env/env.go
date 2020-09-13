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
	"github.com/packing/nbpy/codecs"
	"github.com/packing/nbpy/packets"
	"github.com/packing/nbpy/errors"
	"reflect"
	"sort"
	"github.com/packing/nbpy/utils"
	"os"
	"syscall"
	"os/signal"
	"fmt"
	"runtime"
	"strings"
	"strconv"
)

var collectionCodecs map[byte] *codecs.Codec
var collectionPacketFormats []*packets.PacketFormat

func RegisterCodec(codec *codecs.Codec) (error) {
	k := byte((byte(codec.Protocol) << 4) | byte(codec.Version))
	if collectionCodecs == nil {
		collectionCodecs = make(map[byte] *codecs.Codec)
	}
	utils.LogVerbose(">>> 注册编解码器 封包协议[%d][ver.%d] [%s]", codec.Protocol, codec.Version, codec.Name)
	_, ok := collectionCodecs[k]
	if ok {
		return errors.Errorf("!!! 封包协议[%d][%s] 版本[%d] 已经注册，无需重复注册", codec.Protocol, codec.Version, codec.Name)
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
	return errors.Errorf("!!! 封包协议[%d] 版本[%d] 不存在", protocol, version), nil
}

func RegisterPacketFormat(fmt *packets.PacketFormat) error {
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
		return errors.Errorf("!!! 指定的封包格式 %s 不存在", reflect.TypeOf(*fmt).Kind())
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

func Schedule() os.Signal {

	//定时器实现

	c := make(chan os.Signal, 1)
	signals := []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGSTOP,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGUSR1,
        syscall.SIGUSR2,
	}
	signal.Notify(c, signals...)
	s := <-c
	utils.LogVerbose(">>> 收到信号 > %s", s.String())
	return s
}

func GetGoroutineId() int {
	defer func()  {
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