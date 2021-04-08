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
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/packing/clove/codecs"
	"github.com/packing/clove/errors"
	"github.com/packing/clove/utils"
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
	addr     string
	buffer   *utils.MutexBuffer
	lastTime time.Time
}

type UnixController struct {
	OnStop     OnControllerStop
	id         SessionID
	recvBuffer *utils.MutexBuffer
	sendBuffer map[string]*UnixSendBuffer
	ioinner    net.UnixConn
	DataRW     *DataReadWriter

	queue            chan UnixDatagram
	closeCh          chan int
	sendCh           chan UnixDatagram
	closeOnSended    bool
	closeSendReq     bool
	associatedObject interface{}

	mutex sync.Mutex
	tag   int
}

func createUnixController(ioSrc net.UnixConn, dataRW *DataReadWriter) *UnixController {
	sor := new(UnixController)
	sor.recvBuffer = new(utils.MutexBuffer)
	sor.sendBuffer = make(map[string]*UnixSendBuffer)
	sor.ioinner = ioSrc
	sor.DataRW = dataRW
	sor.id = NewSessionID()
	sor.closeOnSended = false
	sor.associatedObject = nil
	return sor
}

func (receiver *UnixController) SetTag(tag int) {
	receiver.tag = tag
}

func (receiver *UnixController) GetTag() int {
	return receiver.tag
}

func (receiver *UnixController) SetAssociatedObject(o interface{}) {
	receiver.associatedObject = o
}

func (receiver *UnixController) GetAssociatedObject() interface{} {
	return receiver.associatedObject
}

func (receiver UnixController) GetSource() string {
	return receiver.ioinner.LocalAddr().String()
}

func (receiver UnixController) GetSessionID() SessionID {
	return receiver.id
}

func (receiver UnixController) Close() {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()

	close(receiver.sendCh)
	receiver.sendCh = nil

	go func() {
		receiver.closeCh <- 1
	}()

	receiver.ioinner.Close()
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

func (receiver *UnixController) WriteTo(addr string, data []byte) {
	uData := UnixDatagram{addr: addr, data: data}

	go func() {
		receiver.mutex.Lock()
		defer receiver.mutex.Unlock()

		if receiver.sendCh != nil {
			receiver.sendCh <- uData
		}
	}()
}

func (receiver *UnixController) clearSendBuffer(addr string) {
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

func (receiver UnixController) SendFdTo(addr string, fds ...int) error {
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
	defer func() {
		group.Done()
		utils.LogPanic(recover())
	}()

	for {
		datagram, ok := <-receiver.queue
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
}

func (receiver *UnixController) processRead(group *sync.WaitGroup) {
	defer func() {
		group.Done()
		utils.LogPanic(recover())
	}()

	var b = make([]byte, 1024*1024)

	for {
		n, addr, err := receiver.ioinner.ReadFromUnix(b)
		if err != nil {
			break
		}
		IncTotalUnixRecvSize(n)

		pl := receiver.DataRW.PeekPacketLength(b[:n])
		if pl == 0 {
			utils.LogInfo("数据流出错，抛弃数据 => %d", n)
			continue
		}
		if pl == -1 {
			utils.LogInfo("获取到的数据不足以构成完整包，抛弃数据 => %d", n)
			continue
		}

		if n < pl {
			utils.LogInfo("获取到的数据不足以构成完整包，抛弃数据")
			continue
		}

		bs := make([]byte, n)
		copy(bs, b[:n])
		datagram := UnixDatagram{addr: addr.String(), data: bs}
		receiver.queue <- datagram
	}

	close(receiver.queue)
	group.Done()
}

func (receiver *UnixController) innerProcessWrite(uData UnixDatagram) error {

	if len(uData.data) > 1024*1024 {
		utils.LogError(">>> 连接 %s 发送缓冲区数据超过1M 当前 => %d", receiver.GetSource(), len(uData.data))
		return nil
	}

	for {
		//设置写超时，避免客户端一直不收包，导致服务器内存暴涨
		receiver.ioinner.SetWriteDeadline(time.Now().Add(10 * time.Second))
		unixAddr, err := net.ResolveUnixAddr("unixgram", uData.addr)
		if err != nil {
			utils.LogError(">>> 连接 %s 地址解析失败", uData.addr)
			return nil
		}
		_, sendErr := receiver.ioinner.WriteToUnix(uData.data, unixAddr)
		if sendErr == nil {
			IncTotalUnixSendSize(len(uData.data))
			return nil
		}
		if strings.Contains(sendErr.Error(), "sendto: no buffer space available") {
			//utils.LogError(">>> 连接 %s -> %s 发送缓冲区已满，当前数据被抛弃", receiver.GetSource(), uData.addr)
			runtime.Gosched()
			continue
		}
		if strings.Contains(sendErr.Error(), "sendto: connection refused") {
			runtime.Gosched()
			return nil
		}
		if strings.Contains(sendErr.Error(), "sendto: no such file or directory") {
			runtime.Gosched()
			return nil
		}
		if sendErr != nil {
			utils.LogError(">>> 连接 %s -> %s 发送数据超时或异常", receiver.GetSource(), uData.addr)
			utils.LogError(sendErr.Error())
			return sendErr
		}
		break
	}
	return nil

	/*
	   var sendBuffLen uint32 = 0
	   var sendedLen = 0
	   for _, v := range receiver.sendBuffer {

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
	           var b = make([]byte, sendBuffLen)
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
	   } else {
	       IncTotalUnixSendSize(sendedLen)
	   }*/

}

func (receiver *UnixController) processWrite(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		utils.LogPanic(recover())
	}()

main:
	for {
		uData, ok := <-receiver.sendCh
		if ok {
			if err := receiver.innerProcessWrite(uData); err != nil {
				utils.LogError(">>> 连接 %s 发生不可忽略的错误，连接即将被关闭", receiver.GetSource())
				receiver.Close()
				break main
			}

			select {
			case <-receiver.closeCh:
				utils.LogError(">>> 连接 %s 已关闭", receiver.GetSource())
				break main
			default:
			}
		} else {
			utils.LogError(">>> 因连接 %s 关闭，退出数据发送处理", receiver.GetSource())
			break
		}
	}

	utils.LogVerbose(">>> 连接 %s 停止处理I/O发送", receiver.GetSource())
}

func (receiver *UnixController) Schedule() {
	receiver.queue = make(chan UnixDatagram, 1024)
	receiver.closeCh = make(chan int)
	receiver.sendCh = make(chan UnixDatagram, 1024)
	group := new(sync.WaitGroup)
	group.Add(3)

	go func() {
		go receiver.processData(group)
		go receiver.processRead(group)
		go receiver.processWrite(group)
		group.Wait()
		if receiver.OnStop != nil {
			receiver.OnStop(receiver)
		}
		utils.LogError(">>> UNIX控制器 %s 已关闭调度", receiver.GetSource())
	}()
}
