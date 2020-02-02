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
	"nbpy/utils"
	"net"
	"sync"
	"time"
	"strings"
)

/*
goroutine 1 => process wait
goroutine 2 => process read
goroutine 3 => process write
goroutine 4 => process data
*/

type TCPController struct {
	OnStop      OnControllerStop
	id          SessionID
	recvBuffer  *utils.MutexBuffer
	sendBuffer  *utils.MutexBuffer
	ioinner     net.Conn
	DataRW      *DataReadWriter
	runableData chan int
	source      string
	closeCh     chan int
	sendCh     chan int
	closeOnSended bool
	closeSendReq bool
	associatedObject interface{}
}

func createTCPController(ioSrc net.Conn, dataRW *DataReadWriter) *TCPController {
	sor := new(TCPController)
	sor.recvBuffer = new(utils.MutexBuffer)
	sor.sendBuffer = new(utils.MutexBuffer)
	sor.ioinner = ioSrc
	sor.DataRW = dataRW
	sor.source = ioSrc.RemoteAddr().String()
	sor.id = NewSessionID()
	sor.closeOnSended = false
	sor.associatedObject = nil
	return sor
}

func (receiver *TCPController) SetAssociatedObject(o interface{}) {
	receiver.associatedObject = o
}

func (receiver TCPController) GetAssociatedObject() (interface{}) {
	return receiver.associatedObject
}

func (receiver TCPController) GetSource() string {
	return receiver.source
}

func (receiver TCPController) GetSessionID() SessionID {
	return receiver.id
}

func (receiver *TCPController) Close() {
	defer func() {
		utils.LogPanic()
	}()
	receiver.ioinner.Close()
	if receiver.closeCh != nil {
		close(receiver.closeCh)
		receiver.closeCh = nil
	}
	//go func() { receiver.closeCh <- 1 }()
}

func (receiver *TCPController) Discard() {
	receiver.recvBuffer.Reset()
}

func (receiver *TCPController) CloseOnSended() {
	receiver.closeOnSended = true
}

func (receiver *TCPController) Read(l int) ([]byte, int) {
	return receiver.recvBuffer.Next(l)
}

func (receiver *TCPController) Peek(l int) ([]byte, int) {
	return receiver.recvBuffer.Peek(l)
}

func (receiver *TCPController) Write(data []byte) {
	receiver.sendBuffer.Write(data)
	//go func() { receiver.sendCh <- 1 }()
}

func (receiver *TCPController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) {
	//utils.LogVerbose(">>> 连接 %s 发送客户端消息", receiver.GetSource())
	buf, remainMsgs, err := receiver.DataRW.PackStream(receiver, msg...)
	if err == nil {
		receiver.Write(buf)
	}
	return remainMsgs, err
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
	defer func() {
		receiver.Close()
		wg.Done()
		utils.LogPanic()
	}()
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
	utils.LogVerbose(">>> 连接 %s 停止处理数据解析", receiver.GetSource())
}

func (receiver *TCPController) processRead(wg *sync.WaitGroup) {
	defer func() {
		close(receiver.runableData)
		wg.Done()
		utils.LogPanic()
	}()

	var b = make([]byte, 1024)

	utils.LogVerbose(">>> 连接 %s 开始处理I/O读取...", receiver.GetSource())
	for {
		n, err := receiver.ioinner.Read(b)
		if err == nil && n > 0 {
			receiver.recvBuffer.Write(b[:n])
			receiver.runableData <- n
		}
		if err != nil || n == 0 {
			break
		}
	}

	utils.LogVerbose(">>> 连接 %s 停止处理I/O读取", receiver.GetSource())
	receiver.closeSendReq = true
}

func (receiver *TCPController) processWrite(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		utils.LogPanic()
	}()

	for {
		_, ok := <- receiver.sendCh
		if ok {
			var repeat = 0
			buf := make([]byte, 1024)
			sendBuffLen, _ := receiver.sendBuffer.Read(buf)
			for sendBuffLen > 0 {
				//设置写超时，避免客户端一直不收包，导致服务器内存暴涨
				receiver.ioinner.SetWriteDeadline(time.Now().Add(10 * time.Second))
				sizeWrited, sendErr := receiver.ioinner.Write(buf[:sendBuffLen])
				if sendErr == nil && sendBuffLen == sizeWrited {
					if receiver.closeOnSended {
						receiver.Close()
					}
					//utils.LogVerbose(">>> 发送完成", sendBuffLen)
					repeat = 0
					sendBuffLen, _ = receiver.sendBuffer.Read(buf)
					continue
				}
				if sendErr != nil {
					if strings.Contains(sendErr.Error(), "use of closed network connection") {
						break
					}
					if repeat < 1000 {
						utils.LogError(">>> 连接 %s 发送数据超时或异常,重试", receiver.GetSource())
						utils.LogError(sendErr.Error())
						time.Sleep(500 * time.Microsecond)
						repeat += 1
						continue
					} else {
						utils.LogError(">>> 连接 %s 发送数据超时或异常,重试次数已满，关闭连接", receiver.GetSource())
						utils.LogError(sendErr.Error())
						receiver.Close()
					}
				}
				break
			}
		} else {
			utils.LogError(">>> 因连接 %s 关闭，退出数据发送处理", receiver.GetSource())
			break
		}
	}

	utils.LogVerbose(">>> 连接 %s 停止处理I/O发送", receiver.GetSource())
}

func (receiver *TCPController) processSchedule(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		//utils.LogPanic()
	}()
	for {
		if receiver.closeSendReq {
			close(receiver.sendCh)
			break
		}
		receiver.sendCh <- 1
		time.Sleep(1 * time.Millisecond)
	}
}

func (receiver *TCPController) Schedule() {
	receiver.runableData = make(chan int, 10240)
	receiver.closeCh = make(chan int)
	receiver.sendCh = make(chan int, 10240)
	wg := new(sync.WaitGroup)
	wg.Add(4)
	go func() {
		go receiver.processData(wg)
		go receiver.processRead(wg)
		go receiver.processWrite(wg)
		go receiver.processSchedule(wg)
		wg.Wait()
		if receiver.OnStop != nil {
			receiver.OnStop(receiver)
		}
		utils.LogError(">>> TCP控制器 %s 已关闭调度", receiver.GetSource())
	}()
}
