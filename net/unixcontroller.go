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
	"nbpy/utils"
	"sync"
	"net"
	"nbpy/codecs"
	"nbpy/errors"
)

/*
goroutine 1 => process wait
goroutine 2 => process read
goroutine 3 => process data
*/

type UnixDatagram struct {
	addr string
	data []byte
}

type UnixController struct {
	OnStop        OnControllerStop
	id            SessionID
	recvBuffer    *utils.MutexBuffer
	ioinner       net.UnixConn
	DataRW        *DataReadWriter

	queue         chan UnixDatagram
}

func createUnixController(ioSrc net.UnixConn, dataRW *DataReadWriter) (*UnixController) {
	sor := new(UnixController)
	sor.recvBuffer = new(utils.MutexBuffer)
	sor.ioinner = ioSrc
	sor.DataRW = dataRW
	sor.id = NewSessionID()
	return sor
}

func (receiver UnixController) GetSource() (string){
	return receiver.ioinner.LocalAddr().String()
}

func (receiver UnixController) GetSessionID() (SessionID){
	return receiver.id
}

func (receiver UnixController) Close() {
	receiver.ioinner.Close()
}

func (receiver UnixController) Discard() {
}

func (receiver UnixController) Read(l int) ([]byte, int) {
	return nil, 0
}

func (receiver UnixController) Peek(l int) ([]byte, int) {
	return nil, 0
}

func (receiver UnixController) Write(data []byte) {
}

func (receiver UnixController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) {
	return msg, errors.ErrorDataSentIncomplete
}

func (receiver UnixController) ReadFrom() (string, []byte, int) {
	return "", nil, 0
}

func (receiver UnixController) WriteTo(addr string, data []byte) {
	unixAddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		return
	}
	receiver.ioinner.WriteToUnix(data, unixAddr)
}

func (receiver UnixController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) {
	return receiver.DataRW.WriteDatagram(receiver, addr, msg...)
}

func (receiver UnixController) SendFdTo(addr string, fds int) (error) {
	return nil
}

func (receiver *UnixController) processData(group *sync.WaitGroup) {
	defer utils.LogPanic()
	for {
		datagram, ok := <- receiver.queue
		if !ok {
			break
		}
		err := receiver.DataRW.ReadDatagram(receiver, datagram.addr, datagram.data)
		if err != nil {
			receiver.ioinner.Close()
			break
		}
	}

	group.Done()

}
func (receiver *UnixController) processRead(group *sync.WaitGroup) {
	defer utils.LogPanic()
	var b = make([]byte, 1024 * 1024)

	for {
		n, addr, err := receiver.ioinner.ReadFromUnix(b)
		if err != nil {
			break
		}

		bs := make([]byte, n)
		copy(bs, b[:n])
		datagram := UnixDatagram{addr:addr.String(), data: bs}
		receiver.queue <- datagram
	}

	close(receiver.queue)
	group.Done()
}

func (receiver UnixController) Schedule() {
	receiver.queue = make(chan UnixDatagram, 10240)
	go func() {
		group := new(sync.WaitGroup)
		group.Add(2)

		go receiver.processData(group)
		go receiver.processRead(group)

		group.Wait()
		if receiver.OnStop != nil {
			receiver.OnStop(receiver)
		}
	}()
}