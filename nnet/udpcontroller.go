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
	"github.com/packing/nbpy/utils"
	"sync"
	"net"
	"github.com/packing/nbpy/codecs"
	"github.com/packing/nbpy/errors"
)

/*
goroutine 1 => process wait
goroutine 2 => process read
goroutine 3 => process data
*/

type UDPDatagram struct {
	addr string
	data []byte
}

type UDPController struct {
	OnStop        OnControllerStop
	id            SessionID
	recvBuffer    *utils.MutexBuffer
	ioinner       net.UDPConn
	DataRW        *DataReadWriter

	queue         chan UDPDatagram
	associatedObject interface{}
}

func createUDPController(ioSrc net.UDPConn, dataRW *DataReadWriter) (*UDPController) {
	sor := new(UDPController)
	sor.recvBuffer = new(utils.MutexBuffer)
	sor.ioinner = ioSrc
	sor.DataRW = dataRW
	sor.id = NewSessionID()
	sor.associatedObject = nil
	return sor
}

func (receiver *UDPController) SetAssociatedObject(o interface{}) {
	receiver.associatedObject = o
}

func (receiver UDPController) GetAssociatedObject() (interface{}) {
	return receiver.associatedObject
}

func (receiver UDPController) GetSource() (string){
	return receiver.ioinner.LocalAddr().String()
}

func (receiver UDPController) GetSessionID() (SessionID){
	return receiver.id
}

func (receiver UDPController) Close() {
	receiver.ioinner.Close()
}

func (receiver UDPController) Discard() {
}

func (receiver UDPController) CloseOnSended() {
}

func (receiver UDPController) Read(l int) ([]byte, int) {
	return nil, 0
}

func (receiver UDPController) Peek(l int) ([]byte, int) {
	return nil, 0
}

func (receiver UDPController) Write(data []byte) {
}

func (receiver UDPController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) {
	return msg, errors.ErrorDataSentIncomplete
}

func (receiver UDPController) ReadFrom() (string, []byte, int) {
	return "", nil, 0
}

func (receiver UDPController) WriteTo(addr string, data []byte) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return
	}
	receiver.ioinner.WriteToUDP(data, udpAddr)
}

func (receiver UDPController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) {
	buf, remainMsgs, err := receiver.DataRW.PackDatagram(receiver, msg...)
	if err == nil {
		receiver.WriteTo(addr, buf)
	}
	return remainMsgs, err
}

func (receiver *UDPController) processData(group *sync.WaitGroup) {
	defer utils.LogPanic(recover())
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
func (receiver *UDPController) processRead(group *sync.WaitGroup) {
	defer utils.LogPanic(recover())
	var b = make([]byte, 1024 * 1024)

	for {
		n, addr, err := receiver.ioinner.ReadFromUDP(b)
		if err != nil {
			break
		}

		bs := make([]byte, n)
		copy(bs, b[:n])
		datagram := UDPDatagram{addr:addr.String(), data: bs}
		receiver.queue <- datagram
	}

	close(receiver.queue)
	group.Done()
}

func (receiver UDPController) Schedule() {
	receiver.queue = make(chan UDPDatagram, 10240)
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
