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

package messages

import (
    "github.com/packing/nbpy/codecs"
    "github.com/packing/nbpy/errors"
    "github.com/packing/nbpy/nnet"
)

var ErrorDataNotIsMessageMap = errors.Errorf("The data is not message-map")
var ErrorKeyIsRequired = errors.Errorf("The message key is required")

type TagType = int

type Message struct {
    messageSerial    int64
    messageErrorCode int
    messageScheme    int
    messageType      int
    messageSync      bool
    messageTag       codecs.IMSlice
    messageSessionId []nnet.SessionID
    messageBody      codecs.IMMap
    controller       nnet.Controller
    messageSrcData   codecs.IMData
    addr             string
    unixAddr         string
}

func MessageFromData(controller nnet.Controller, addr string, data codecs.IMData) (*Message, error) {
    mapData, ok := data.(codecs.IMMap)
    if !ok {
        return nil, ErrorDataNotIsMessageMap
    }

    reader := codecs.CreateMapReader(mapData)

    msg := new(Message)
    msg.messageType = int(reader.IntValueOf(ProtocolKeyType, 0))
    msg.messageScheme = int(reader.IntValueOf(ProtocolKeyScheme, 0))
    msg.messageErrorCode = int(reader.IntValueOf(ProtocolKeyErrorCode, ProtocolErrorCodeOK))
    msg.messageSync = reader.BoolValueOf(ProtocolKeySync)
    msg.messageSerial = reader.IntValueOf(ProtocolKeySerial, 0)
    msg.unixAddr = reader.StrValueOf(ProtocolKeyUnixAddr, "")

    msg.messageSessionId = make([]nnet.SessionID, 0)
    sessIds := reader.TryReadValue(ProtocolKeySessionId)
    if sessIds != nil {
        sess, _ := sessIds.(codecs.IMSlice)
        if ok {
            msg.messageSessionId = make([]nnet.SessionID, len(sess))
            for i, sid := range sess {
                ssid := codecs.Int64FromInterface(sid)
                msg.messageSessionId[i] = nnet.SessionID(ssid)
            }
        }
    }

    msg.messageTag = make(codecs.IMSlice, 0)
    itagss := reader.TryReadValue(ProtocolKeyTag)
    if itagss != nil {
        itags, ok := itagss.(codecs.IMSlice)
        if ok {
            for _, tag := range itags {
                msg.messageTag = append(msg.messageTag, tag)
            }
        }
    }

    msg.messageBody = nil
    ibody := reader.TryReadValue(ProtocolKeyBody)
    if ibody != nil {
        body, ok := ibody.(codecs.IMMap)
        if ok {
            msg.messageBody = body
        }
    }

    msg.controller = controller
    msg.addr = addr
    msg.messageSrcData = data
    return msg, nil
}

func DataFromMessage(message *Message) (codecs.IMData, error) {
    msg := make(codecs.IMMap)
    msg[ProtocolKeyScheme] = message.messageScheme
    msg[ProtocolKeyType] = message.messageType
    msg[ProtocolKeyTag] = message.messageTag
    msg[ProtocolKeySync] = message.messageSync

    ssid := make(codecs.IMSlice, 0)
    for _, sid := range message.messageSessionId {
        ssid = append(ssid, sid)
    }

    msg[ProtocolKeySessionId] = ssid
    msg[ProtocolKeyBody] = message.messageBody
    msg[ProtocolKeyErrorCode] = message.messageErrorCode
    msg[ProtocolKeySerial] = message.messageSerial
    return msg, nil
}

func CreateMessage(errorCode, scheme, mtype int, tag codecs.IMSlice, sync bool, sessid []nnet.SessionID, body codecs.IMMap) (*Message, error) {
    msg := new(Message)
    msg.messageScheme = scheme
    msg.messageType = mtype
    msg.messageTag = tag
    msg.messageSync = sync
    msg.messageSessionId = sessid
    msg.messageBody = body
    msg.messageErrorCode = errorCode
    msg.messageSerial = 0
    return msg, nil
}

func (receiver Message) GetUnixSource() string {
    return receiver.unixAddr
}

func (receiver Message) GetSource() string {
    return receiver.addr
}

func (receiver Message) GetController() nnet.Controller {
    return receiver.controller
}

func (receiver Message) GetSrcData() codecs.IMData {
    return receiver.messageSrcData
}

func (receiver *Message) SetSearial(searial int64) {
    receiver.messageSerial = searial
}

func (receiver Message) GetSearial() int64 {
    return receiver.messageSerial
}

func (receiver *Message) SetErrorCode(code int) {
    receiver.messageErrorCode = code
}

func (receiver Message) GetErrorCode() int {
    return receiver.messageErrorCode
}

func (receiver *Message) SetScheme(scheme int) {
    receiver.messageScheme = scheme
}

func (receiver Message) GetScheme() int {
    return receiver.messageScheme
}

func (receiver *Message) SetType(tp int) {
    receiver.messageType = tp
}

func (receiver Message) GetType() int {
    return receiver.messageType
}

func (receiver *Message) SetTag(tag TagType) {
    if receiver.messageTag == nil {
        receiver.messageTag = make(codecs.IMSlice, 0)
    }
    for sTag := range receiver.messageTag {
        if sTag == tag {
            break
        }
    }
    receiver.messageTag = append(receiver.messageTag, tag)
}

func (receiver *Message) UnsetTag(tag TagType) {
    if receiver.messageTag == nil {
        receiver.messageTag = make(codecs.IMSlice, 0)
    }
    ts := make(codecs.IMSlice, 0)
    for sTag := range receiver.messageTag {
        if sTag != tag {
            ts = append(ts, sTag)
        }
    }
    receiver.messageTag = ts
}

func (receiver Message) GetTag() codecs.IMSlice {
    return receiver.messageTag
}

func (receiver *Message) SetSync(sync bool) {
    receiver.messageSync = sync
}

func (receiver Message) GetSync() bool {
    return receiver.messageSync
}

func (receiver *Message) SetSessionId(sessid []nnet.SessionID) {
    receiver.messageSessionId = sessid
}

func (receiver Message) GetSessionId() []nnet.SessionID {
    return receiver.messageSessionId
}

func (receiver *Message) SetBody(body codecs.IMMap) {
    receiver.messageBody = body
}

func (receiver Message) GetBody() codecs.IMMap {
    return receiver.messageBody
}

func CreateS2SMessage(tp int) *Message {
    msg := new(Message)
    msg.messageType = tp
    msg.messageScheme = ProtocolSchemeS2S
    return msg
}

func CreateC2SMessage(tp int) *Message {
    msg := new(Message)
    msg.messageType = tp
    msg.messageScheme = ProtocolSchemeC2S
    return msg
}

func CreateS2CMessage(tp int) *Message {
    msg := new(Message)
    msg.messageType = tp
    msg.messageScheme = ProtocolSchemeS2C
    return msg
}

func CreateC2SReturnMessage(c2sMsg *Message) *Message {
    msg := new(Message)
    msg.messageType = c2sMsg.messageType
    msg.messageScheme = ProtocolSchemeS2C
    msg.SetSessionId(c2sMsg.GetSessionId())
    msg.messageSerial = c2sMsg.messageSerial
    return msg
}
