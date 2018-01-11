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
	"nbpy/codecs"
	"nbpy/net"
	"nbpy/errors"
)

var ErrorDataNotIsMessageMap = errors.Errorf("The data is not message-map")
var ErrorKeyIsRequired = errors.Errorf("The message key is required")

type TagType = int

type Message struct {
	messageType int
	messageSync bool
	messageTag []TagType
	messageSessionId []net.SessionID
	messageBody codecs.IMMap
}

func CreateMessage(data codecs.IMData) (*Message, error) {
	mapData, ok := data.(codecs.IMMap)
	if !ok {
		return nil, ErrorDataNotIsMessageMap
	}

	tp, ok := mapData[ProtocolKeyType]
	if !ok {
		return nil, ErrorKeyIsRequired
	}

	intTp, ok := tp.(int)
	if !ok {
		return nil, ErrorKeyIsRequired
	}

	msg := new(Message)
	msg.messageType = intTp

	sync, ok := mapData[ProtocolKeySync]
	if ok {
		bsync, ok := sync.(bool)
		msg.messageSync = ok && bsync
	}

	msg.messageSessionId = make([]net.SessionID,0)
	sessIds, ok := mapData[ProtocolKeySessionId]
	if ok {
		sess, ok := sessIds.(codecs.IMSlice)
		if ok {
			msg.messageSessionId = make([]net.SessionID,len(sess))
			for i, sid := range sess {
				ssid ,ok := sid.(int)
				if ok {
					msg.messageSessionId[i] = net.SessionID(ssid)
				}
			}
		}
	}

	msg.messageTag = make([]TagType, 0)
	itagss, ok := mapData[ProtocolKeyTag]
	if ok {
		itags, ok := itagss.(codecs.IMSlice)
		if ok {
			msg.messageTag = make([]TagType,len(itags))
			for i, itag := range itags {
				tag ,ok := itag.(int)
				if ok {
					msg.messageTag[i] = tag
				}
			}
		}
	}

	msg.messageBody = nil
	ibody, ok := mapData[ProtocolKeyBody]
	if ok {
		body, ok := ibody.(codecs.IMMap)
		if ok {
			msg.messageBody = body
		}
	}

	return msg, nil
}