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

package utils

import (
	"bytes"
	"sync"
)

type MutexBuffer struct {
	b  bytes.Buffer
	rw sync.Mutex
}

func (b *MutexBuffer) Read(p []byte) (int, error) {
	b.rw.Lock()
	defer b.rw.Unlock()
	return b.b.Read(p)
}

func (b *MutexBuffer) Next(n int) ([]byte, int) {
	b.rw.Lock()
	defer b.rw.Unlock()
	var sn = n
	var bl = b.b.Len()
	if bl < n {
		sn = bl
	}
	return b.b.Next(sn), sn
}

func (b *MutexBuffer) Peek(n int) ([]byte, int) {
	b.rw.Lock()
	defer b.rw.Unlock()
	var sn = n
	var bl = b.b.Len()
	if bl < n {
		sn = bl
	}
	//var r = make([]byte, sn)
	//copy(r, b.b.Bytes()[:sn])
	return b.b.Bytes()[:sn], sn
}

func (b *MutexBuffer) Write(p []byte) (int, error) {
	b.rw.Lock()
	defer b.rw.Unlock()
	return b.b.Write(p)
}

func (b *MutexBuffer) Reset() {
	b.rw.Lock()
	defer b.rw.Unlock()
	b.b.Reset()
}

func (b *MutexBuffer) Len() int {
	b.rw.Lock()
	defer b.rw.Unlock()
	return b.b.Len()
}
