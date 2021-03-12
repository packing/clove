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
    "github.com/packing/nbpy/codecs"
    "math"
    "sync"
)

type Controller interface {
    GetSource() string
    GetSessionID() SessionID
    Discard()
    Read(int) ([]byte, int)
    Peek(int) ([]byte, int)
    Write([]byte)
    Send(...codecs.IMData) ([]codecs.IMData, error)
    ReadFrom() (string, []byte, int)
    WriteTo(string, []byte)
    SendTo(string, ...codecs.IMData) ([]codecs.IMData, error)
    Close()
    Schedule()
    CloseOnSended()
    GetAssociatedObject() interface{}
    GetTag() int
    SetTag(int)
}

type SocketController struct {
    OnWelcome func(Controller) error
    OnBye     func(Controller) error
}

type OnControllerStop func(Controller) error
type OnControllerCome func(Controller) error

type Server interface {
    Lookup()
    Boardcast(codecs.IMData)
    Mutilcast([]SessionID, codecs.IMData)
}

type FileHandleController interface {
    OnFileHandleReceived(fd int) error
}

type SessionID = uint64

var mutex sync.Mutex
var mutex2 sync.Mutex

var wsCodecDefault = codecs.CodecIMv2

var currentSessionId SessionID = 0

var sendbufferSize = 1024
var recvbufferSize = 1024

var totalTcpSendSize = 0
var totalTcpRecvSize = 0

var totalUnixSendSize = 0
var totalUnixRecvSize = 0

var totalHandleRecvSize = 0

var decodeInstanceCount = 0

var encodeTime int64 = 0
var decodeTime int64 = 0
var encodeCount int64 = 0
var decodeCount int64 = 0

func SetWebsocketDefaultCodec(codec *codecs.Codec) {
    wsCodecDefault = codec
}

func IncEncodeTime(tv int64) {
    encodeTime += tv
    encodeCount += 1
}

func IncDecodeTime(tv int64) {
    decodeTime += tv
    decodeCount += 1
}

func GetEncodeAgvTime() int64 {
    if encodeCount > 0 {
        return encodeTime / encodeCount
    }
    return 0
}

func GetDecodeAgvTime() int64 {
    if decodeCount > 0 {
        return decodeTime / decodeCount
    }
    return 0
}

func GetDecodeInstanceCount() int {
    mutex2.Lock()
    defer mutex2.Unlock()
    return decodeInstanceCount
}

func IncDecodeInstanceCount() {
    mutex2.Lock()
    defer mutex2.Unlock()
    decodeInstanceCount += 1
}

func DecDecodeInstanceCount() {
    mutex2.Lock()
    defer mutex2.Unlock()
    decodeInstanceCount -= 1
}

func IncTotalTcpSendSize(s int) {
    totalTcpSendSize += s
}

func IncTotalTcpRecvSize(s int) {
    totalTcpRecvSize += s
}

func GetTotalTcpSendSize() int {
    return totalTcpSendSize
}

func GetTotalTcpRecvSize() int {
    return totalTcpRecvSize
}

func IncTotalUnixSendSize(s int) {
    totalUnixSendSize += s
}

func IncTotalUnixRecvSize(s int) {
    totalUnixRecvSize += s
}

func GetTotalUnixSendSize() int {
    return totalUnixSendSize
}

func GetTotalUnixRecvSize() int {
    return totalUnixRecvSize
}

func IncTotalHandleRecvSize(s int) {
    totalHandleRecvSize += s
}

func GetTotalHandleSendSize() int {
    return totalHandleRecvSize
}

func SetSendBufSize(s int) {
    sendbufferSize = s
}

func SetRecvBufSize(s int) {
    recvbufferSize = s
}

func NewSessionID() SessionID {
    mutex.Lock()
    defer mutex.Unlock()
    currentSessionId += 1
    if currentSessionId >= math.MaxUint64 {
        currentSessionId = 1
    }
    return currentSessionId
}
