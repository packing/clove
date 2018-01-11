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
)

/*
goroutine 1 => process wait
goroutine 2 => process read
goroutine 3 => process data
*/

type TCPController struct {
	OnStop        OnControllerStop
	id            SessionID
	recvBuffer    *utils.MutexBuffer
	ioinner       net.Conn
	DataRW        *DataReadWriter
	runableData   chan int
	source        string
}

func createTCPController(ioSrc net.Conn, dataRW *DataReadWriter) (*TCPController) {
	sor := new(TCPController)
	sor.recvBuffer = new(utils.MutexBuffer)
	sor.ioinner = ioSrc
	sor.DataRW = dataRW
	sor.source = ioSrc.RemoteAddr().String()
	sor.id = NewSessionID()
	return sor
}

func (receiver TCPController) GetSource() (string){
	return receiver.source
}

func (receiver TCPController) GetSessionID() (SessionID){
	return receiver.id
}

func (receiver TCPController) Close() {
	receiver.ioinner.Close()
}

func (receiver TCPController) Discard() {
	receiver.recvBuffer.Reset()
}

func (receiver TCPController) Read(l int) ([]byte, int) {
	return receiver.recvBuffer.Next(l)
}

func (receiver TCPController) Peek(l int) ([]byte, int) {
	return receiver.recvBuffer.Peek(l)
}

func (receiver TCPController) Write(data []byte) {
	wn, err := receiver.ioinner.Write(data)
	if err == nil && wn == len(data) {
	}
	if err != nil {
		return
	}
}

func (receiver TCPController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) {
	utils.LogVerbose(">>> 连接 %s 发送客户端消息", receiver.GetSource())
	return receiver.DataRW.WriteStream(receiver, msg...)
}

func (receiver TCPController) ReadFrom() (string, []byte, int) {
	return "", nil, 0
}

func (receiver TCPController) WriteTo(addr string, data []byte) {

}

func (receiver TCPController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) {
	return nil, nil
}

func (receiver *TCPController) processData(wg *sync.WaitGroup) {
	defer utils.LogPanic()
	utils.LogVerbose(">>> 连接 %s 开始处理数据解析...", receiver.GetSource())
	for {
		n, ok := <- receiver.runableData
		if !ok {
			break
		}
		if n == 0 {
			continue
		}
		err := receiver.DataRW.ReadStream(receiver)
		if err != nil {
			receiver.Close()
			break
		}
	}
	wg.Done()
	utils.LogVerbose(">>> 连接 %s 停止处理数据解析", receiver.GetSource())
}

func (receiver *TCPController) processRead(wg *sync.WaitGroup) {
	defer utils.LogPanic()

	var b = make([]byte, 1024)

	utils.LogVerbose(">>> 连接 %s 开始处理I/O读取...", receiver.GetSource())
	for {
		n, err := receiver.ioinner.Read(b)
		if err == nil && n > 0 {
			receiver.recvBuffer.Write(b[:n])
			receiver.runableData <- n
		}
		if err != nil {
			break
		}
	}

	close(receiver.runableData)
	wg.Done()
	utils.LogVerbose(">>> 连接 %s 停止处理I/O读取", receiver.GetSource())
}

func (receiver TCPController) Schedule() {
	receiver.runableData = make(chan int, 10240)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		go receiver.processData(wg)
		go receiver.processRead(wg)
		wg.Wait()
		if receiver.OnStop != nil {
			receiver.OnStop(receiver)
		}
	}()
}