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

package net

import (
	"nbpy/codecs"
	"sync"
	"time"
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
}

type SocketController struct {
	OnWelcome func(Controller) error
	OnBye func(Controller) error
}

type OnControllerStop func(Controller) error

type Server interface {
	Lookup()
	Boardcast(codecs.IMData)
	Mutilcast([]SessionID, codecs.IMData)
}

type SessionID = uint64
var mutex sync.Mutex

func NewSessionID() SessionID {
	mutex.Lock()
	defer mutex.Unlock()
	s := uint64(time.Now().UnixNano())
	s = s & 0x7FFFFFFF
	//time.Sleep(100 * time.Nanosecond)
	return s
}