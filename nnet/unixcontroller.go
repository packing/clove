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
	"time"
	"encoding/binary"
	"strings"
	"syscall"
	"unsafe"
	"runtime"
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

type UnixSendBuffer struct {
	addr 		string
	buffer    	*utils.MutexBuffer
	lastTime  	time.Time
}

type UnixController struct {
	OnStop        OnControllerStop
	id            SessionID
	recvBuffer    *utils.MutexBuffer
	sendBuffer    map[string] *UnixSendBuffer
	ioinner       net.UnixConn
	DataRW        *DataReadWriter

	queue         chan UnixDatagram
	closeCh       chan int
	sendCh        chan int
	closeOnSended bool
	closeSendReq  bool
	associatedObject interface{}
}

func createUnixController(ioSrc net.UnixConn, dataRW *DataReadWriter) (*UnixController) {
	sor := new(UnixController)
	sor.recvBuffer = new(utils.MutexBuffer)
	sor.sendBuffer = make(map[string] *UnixSendBuffer)
	sor.ioinner = ioSrc
	sor.DataRW = dataRW
	sor.id = NewSessionID()
	sor.closeOnSended = false
	sor.associatedObject = nil
	return sor
}

func (receiver *UnixController) SetAssociatedObject(o interface{}) {
	receiver.associatedObject = o
}

func (receiver *UnixController) GetAssociatedObject() (interface{}) {
	return receiver.associatedObject
}

func (receiver UnixController) GetSource() (string){
	return receiver.ioinner.LocalAddr().String()
}

func (receiver UnixController) GetSessionID() (SessionID){
	return receiver.id
}

func (receiver UnixController) Close() {
	receiver.ioinner.Close()
	go func() {
		receiver.closeCh <- 1
	}()
}

func (receiver UnixController) Discard() {
}

func (receiver UnixController) CloseOnSended() {
	receiver.closeOnSended = true
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

var mutexSend sync.Mutex

func (receiver *UnixController) WriteTo(addr string, data []byte) {
	mutexSend.Lock()
	defer mutexSend.Unlock()
	v, ok := receiver.sendBuffer[addr]
	if !ok {
		v = new(UnixSendBuffer)
		v.buffer = new(utils.MutexBuffer)
		v.addr = addr
		v.lastTime = time.Now()
		receiver.sendBuffer[addr] = v
	}
	var b = make([]byte, 4)
	l := uint32(len(data))
	binary.BigEndian.PutUint32(b, l)
	v.buffer.Write(b)
	v.buffer.Write(data)

	go func() {
        receiver.sendCh <- 1
    }()
}

func (receiver *UnixController) clearSendBuffer(addr string) {
	//mutexSend.Lock()
	//defer mutexSend.Unlock()
	v, ok := receiver.sendBuffer[addr]
	if ok {
		v.buffer.Reset()
	}
}

func (receiver *UnixController) SendTo(addr string, msg ...codecs.IMData) ([]codecs.IMData, error) {
    st := time.Now().UnixNano()
	buf, remainMsgs, err := receiver.DataRW.PackDatagram(receiver, msg...)
    IncEncodeTime(time.Now().UnixNano() - st)
	if err == nil {
		receiver.WriteTo(addr, buf)
	} else {
        utils.LogInfo(">>> 请求发送数据时编码器返回错误", err)
    }
	return remainMsgs, err
}

func (receiver UnixController) SendFdTo(addr string, fds...int) (error) {
	unixAddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		return err
	}
	b := syscall.UnixRights(fds...)
	h := (*syscall.Cmsghdr)(unsafe.Pointer(&b[0]))
	oob := b[syscall.CmsgLen(0):h.Len]
	_, _, err = receiver.ioinner.WriteMsgUnix(b, oob, unixAddr)
	return err
}

func (receiver *UnixController) processData(group *sync.WaitGroup) {
	defer utils.LogPanic(recover())
	for {
		datagram, ok := <- receiver.queue
		if !ok {
			break
		}
        st := time.Now().UnixNano()
		err := receiver.DataRW.ReadDatagram(receiver, datagram.addr, datagram.data)
        IncDecodeTime(time.Now().UnixNano() - st)
		if err != nil {
			receiver.ioinner.Close()
			break
		}
	}

	group.Done()
}

func (receiver *UnixController) processRead(group *sync.WaitGroup) {
	defer utils.LogPanic(recover())
	var b = make([]byte, 1024 * 1024)

	for {
		n, addr, err := receiver.ioinner.ReadFromUnix(b)
		if err != nil {
			break
		}
        IncTotalUnixRecvSize(n)

		bs := make([]byte, n)
		copy(bs, b[:n])
		datagram := UnixDatagram{addr:addr.String(), data: bs}
		receiver.queue <- datagram
	}

	close(receiver.queue)
	group.Done()
}

func (receiver *UnixController) innerProcessWrite() (bool) {
	var sendBuffLen uint32 = 0
	var sendedLen = 0
	for _,v := range receiver.sendBuffer {

		var ret = func() (bool) {
			mutexSend.Lock()
			defer mutexSend.Unlock()

			if v.buffer.Len() < 4 {
				return false
			}
			var bb = make([]byte, 4)
			v.buffer.Read(bb)
			sendBuffLen = binary.BigEndian.Uint32(bb)
			if sendBuffLen > 1024*1024 {
				//receiver.Close()
				utils.LogError(">>> 连接 %s 发送缓冲区数据超过1M %d", receiver.GetSource(), sendBuffLen)
				//delete(receiver.sendBuffer, k)
				receiver.clearSendBuffer(v.addr)
				return true
			}
			var b= make([]byte, sendBuffLen)
			var osize = len(b)
			if sendBuffLen > 0 {
				size, err := v.buffer.Read(b)
				if err == nil {
					//设置写超时，避免客户端一直不收包，导致服务器内存暴涨
					receiver.ioinner.SetWriteDeadline(time.Now().Add(10 * time.Second))

				sendProc:

					unixAddr, err := net.ResolveUnixAddr("unixgram", v.addr)
					if err != nil {
						receiver.clearSendBuffer(v.addr)
						utils.LogError(">>> 连接 %s 地址解析失败", v.addr)
						return true
					}
					_, sendErr := receiver.ioinner.WriteToUnix(b[:size], unixAddr)
					if sendErr == nil {
						sendedLen += size
						return false
					}
					if sendErr == nil && size < osize {
						if receiver.closeOnSended {
							receiver.Close()
						}
						receiver.clearSendBuffer(v.addr)
						return true
					}
					if strings.Contains(sendErr.Error(), "sendto: no buffer space available") {
						time.Sleep(1 * time.Millisecond)
						goto sendProc
					}
					if sendErr != nil {
						utils.LogError(">>> 连接 %s -> %s 发送数据超时或异常", receiver.GetSource(), v.addr)
						utils.LogError(sendErr.Error())
						receiver.clearSendBuffer(v.addr)
						return true
					}

				} else {
					return false
				}
				v.lastTime = time.Now()
			}
			return false
		}()

		if ret {
			return true
		}
	}
	if sendedLen == 0 {
		runtime.Gosched()
		//time.Sleep(1 * time.Millisecond)
	} else {
		IncTotalUnixSendSize(sendedLen)
	}
	return false
}

func (receiver *UnixController) processWrite(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		utils.LogPanic(recover())
	}()

	for {
		_, ok := <- receiver.sendCh
		if ok {
			if receiver.innerProcessWrite() {
				time.Sleep(1 * time.Millisecond)
			}
			b := false
			select {
			case <-receiver.closeCh:
				utils.LogError(">>> 连接 %s 已关闭", receiver.GetSource())
				b = true
			default:
				b = false
			}
			if b {
				break
			} else {
				//time.Sleep(10 * time.Millisecond)
			}
		} else {
			utils.LogError(">>> 因连接 %s 关闭，退出数据发送处理", receiver.GetSource())
			break
		}
	}

	utils.LogVerbose(">>> 连接 %s 停止处理I/O发送", receiver.GetSource())
}

func (receiver *UnixController) processSchedule(wg *sync.WaitGroup) {
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
		time.Sleep(10 * time.Millisecond)
	}
}

func (receiver *UnixController) Schedule() {
	receiver.queue = make(chan UnixDatagram, 10240)
	receiver.closeCh = make(chan int)
	receiver.sendCh = make(chan int)
	group := new(sync.WaitGroup)
	group.Add(3)

	go func() {
		go receiver.processData(group)
		go receiver.processRead(group)
		go receiver.processWrite(group)
		//go receiver.processSchedule(group)
		group.Wait()
		if receiver.OnStop != nil {
			receiver.OnStop(receiver)
		}
		utils.LogError(">>> UNIX控制器 %s 已关闭调度", receiver.GetSource())
	}()
}