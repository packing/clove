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
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/packing/clove/codecs"
	"github.com/packing/clove/errors"
	"github.com/packing/clove/utils"
)

/*
goroutine 1 => process wait
goroutine 2 => process read
goroutine 3 => process write
goroutine 4 => process data
*/

type TCPController struct {
	OnStop           OnControllerStop
	id               SessionID
	recvBuffer       *utils.MutexBuffer
	sendBuffer       *utils.MutexBuffer
	ioinner          net.Conn
	DataRW           *DataReadWriter
	runableData      chan int
	source           string
	flowCh           chan int
	sendCh           chan int
	closeOnSended    bool
	closeSendReq     bool
	flowMode         bool
	flowQueueLimit   int
	associatedObject interface{}
	mutex            sync.Mutex
	tag              int
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
	sor.flowCh = nil
	return sor
}

func (receiver *TCPController) SetAssociatedObject(o interface{}) {
	receiver.associatedObject = o
}

func (receiver TCPController) GetAssociatedObject() interface{} {
	return receiver.associatedObject
}

func (receiver *TCPController) SetFlowQueueLimit(limit int) {
	receiver.flowQueueLimit = limit
}

func (receiver *TCPController) SetFlowMode(b bool) {
	receiver.flowMode = b
	if b {
		if receiver.flowCh == nil {
			receiver.flowCh = make(chan int)
			receiver.UnlockProcess()
		}
	} else {
		if receiver.flowCh != nil {
			close(receiver.flowCh)
			receiver.flowCh = nil
		}
	}
}

func (receiver TCPController) IsFlowMode() bool {
	return receiver.flowMode
}

func (receiver *TCPController) LockProcess() bool {
	if receiver.flowMode {
		_, ok := <-receiver.flowCh
		if !ok {
			return false
		}
	}
	return true
}

func (receiver *TCPController) UnlockProcess() {
	if receiver.flowMode {
		go func() {
			receiver.flowCh <- 1
		}()
	}
}

func (receiver *TCPController) SetTag(tag int) {
	receiver.tag = tag
}

func (receiver *TCPController) GetTag() int {
	return receiver.tag
}

func (receiver TCPController) GetSource() string {
	return receiver.source
}

func (receiver TCPController) GetSessionID() SessionID {
	return receiver.id
}

func (receiver *TCPController) Close() {
	receiver.mutex.Lock()
	defer func() {
		receiver.mutex.Unlock()
		utils.LogPanic(recover())
	}()
	receiver.closeSendReq = true
	if receiver.sendCh != nil {
		close(receiver.sendCh)
		receiver.sendCh = nil
	}

	receiver.SetFlowMode(false)
	receiver.ioinner.Close()
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
	if receiver.closeSendReq {
		return
	}
	receiver.sendBuffer.Write(data)

	go func() {
		receiver.mutex.Lock()
		defer func() {
			receiver.mutex.Unlock()
		}()
		if receiver.closeSendReq {
			return
		}
		receiver.sendCh <- 1
	}()
}

func (receiver *TCPController) Send(msg ...codecs.IMData) ([]codecs.IMData, error) {
	//utils.LogVerbose(">>> 连接 %s 发送客户端消息", receiver.GetSource())
	if receiver.closeSendReq {
		return msg, errors.ErrorRemoteReqClose
	}
	st := time.Now().UnixNano()
	buf, remainMsgs, err := receiver.DataRW.PackStream(receiver, msg...)
	utils.LogInfo("Tcp send encode: ", err, len(buf))
	IncEncodeTime(time.Now().UnixNano() - st)
	if err == nil {
		receiver.Write(buf)
	}
	return remainMsgs, err
}

func (receiver *TCPController) RawSend(msg ...codecs.IMData) error {
	//utils.LogVerbose(">>> 连接 %s 发送客户端消息", receiver.GetSource())
	if receiver.closeSendReq {
		return errors.ErrorRemoteReqClose
	}
	st := time.Now().UnixNano()
	buf, _, err := receiver.DataRW.PackStream(receiver, msg...)
	IncEncodeTime(time.Now().UnixNano() - st)
	if err == nil {
	sendProc:
		receiver.ioinner.SetWriteDeadline(time.Now().Add(3 * time.Second))
		n, sendErr := receiver.ioinner.Write(buf)
		if sendErr == nil {
			IncTotalTcpSendSize(n)
			if n < len(buf) {
				buf = buf[n:]
				goto sendProc
			} else {
			}
		} else {
			utils.LogInfo("RawSend Fail!!! => ", sendErr)
			return sendErr
		}
	}
	return nil
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
		wg.Done()
		utils.LogPanic(recover())
	}()
	//utils.LogVerbose(">>> 连接 %s 开始处理数据解析...", receiver.GetSource())

	for {
		n, ok := <-receiver.runableData
		if !ok {
			break
		}
		if n == 0 {
			continue
		}

		ok = receiver.LockProcess()
		if !ok {
			break
		}
		if receiver.flowMode && len(receiver.runableData) > receiver.flowQueueLimit {
			utils.LogInfo(">>> 连接 %s 流处理队列长度超出限制，将被强行关闭", receiver.GetSource())
			receiver.UnlockProcess()
			receiver.Close()
			break
		}

		st := time.Now().UnixNano()
		err := receiver.DataRW.ReadStream(receiver, receiver.recvBuffer)
		IncDecodeTime(time.Now().UnixNano() - st)

		if err != nil {
			receiver.Close()
			break
		}
		runtime.Gosched()

	}
	utils.LogVerbose(">>> 连接 %s 停止处理数据解析", receiver.GetSource())
}

func (receiver *TCPController) processRead(wg *sync.WaitGroup) {
	defer func() {
		close(receiver.runableData)
		wg.Done()
		utils.LogPanic(recover())
	}()

	var b = make([]byte, recvbufferSize)

	//utils.LogVerbose(">>> 连接 %s 开始处理I/O读取...", receiver.GetSource())
	for {
		n, err := receiver.ioinner.Read(b)
		if err == nil && n > 0 {
			IncTotalTcpRecvSize(n)
			receiver.recvBuffer.Write(b[:n])
			receiver.runableData <- n
			runtime.Gosched()
		}
		if err != nil || n == 0 {
			receiver.Close()
			break
		}
	}

	utils.LogVerbose(">>> 连接 %s 停止处理I/O读取", receiver.GetSource())
}

func (receiver *TCPController) processWrite(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		utils.LogPanic(recover())
	}()

	for {
		_, ok := <-receiver.sendCh
		if ok && !receiver.closeSendReq {
			buf := make([]byte, sendbufferSize)

		main:
			sendBuffLen, _ := receiver.sendBuffer.Read(buf)
			tobuf := buf[:sendBuffLen]
			for sendBuffLen > 0 {
				//设置写超时，避免客户端一直不收包，导致服务器内存暴涨
				receiver.ioinner.SetWriteDeadline(time.Now().Add(3 * time.Second))
				sizeWrited, sendErr := receiver.ioinner.Write(tobuf)
				if sendErr == nil {
					if sendBuffLen == sizeWrited {
						if receiver.closeOnSended {
							receiver.Close()
						}

						IncTotalTcpSendSize(sendBuffLen)
						runtime.Gosched()
						goto main
					} else {
						tobuf = tobuf[:sizeWrited]
						sendBuffLen = sendBuffLen - sizeWrited
						IncTotalTcpSendSize(sizeWrited)
						continue
					}
				}
				if sendErr != nil {
					utils.LogError(">>> 连接 %s 发送数据超时或异常，关闭连接", receiver.GetSource())
					utils.LogError(sendErr.Error())
					receiver.Close()
				}
				break
			}
		} else {
			break
		}
	}

	utils.LogVerbose(">>> 连接 %s 停止处理I/O发送", receiver.GetSource())
}

func (receiver *TCPController) Schedule() {
	receiver.runableData = make(chan int, 1024)
	receiver.sendCh = make(chan int)
	wg := new(sync.WaitGroup)
	wg.Add(3)
	go func() {
		go receiver.processData(wg)
		go receiver.processRead(wg)
		go receiver.processWrite(wg)
		wg.Wait()
		if receiver.OnStop != nil {
			receiver.OnStop(receiver)
		}
		utils.LogVerbose(">>> TCP控制器 %s 已关闭调度", receiver.GetSource())
	}()
}
