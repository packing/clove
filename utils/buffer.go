package utils

import (
	"bytes"
	"sync"
)

type MutexBuffer struct {
	b bytes.Buffer
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