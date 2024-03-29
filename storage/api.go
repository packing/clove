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

package storage

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/packing/clove/codecs"
	"github.com/packing/clove/errors"
	"github.com/packing/clove/messages"
	"github.com/packing/clove/nnet"
	"github.com/packing/clove/packets"
	"github.com/packing/clove/utils"
)

var CmdErrorRet = errors.Errorf("storage timeout")

const (
	TXActionNothing = iota
	TXActionInsert
	TXActionUpdate
)

type Transaction struct {
	sql    string
	action int
	args   []interface{}
}

type ResultWaiter struct {
	ch chan interface{}
	tr *time.Timer
	id int64
}

type Client struct {
	udpUnix   *nnet.UnixUDP
	tcpNormal *nnet.TCPClient
	addr      string
	unixMode  bool
	lock      sync.Mutex
	timeOut   time.Duration
	waiters   *sync.Map
	uniqueId  int64
}

func CreateClientWithBufferSize(addr string, timeOut time.Duration, buffWriteSize int, buffReadSize int) *Client {
	kv := new(Client)
	kv.timeOut = timeOut
	kv.waiters = new(sync.Map)
	if err := kv.Initialize(addr, buffWriteSize, buffReadSize); err != nil {
		utils.LogError("CreateClient error: %s", err.Error())
		return nil
	}
	return kv
}

func CreateClient(addr string, timeOut time.Duration) *Client {
	return CreateClientWithBufferSize(addr, timeOut, -1, -1)
}

func (receiver *Client) Close() {
	receiver.waiters.Range(func(key, value interface{}) bool {
		v, ok := value.(ResultWaiter)
		if ok {
			close(v.ch)
		}
		return true
	})

	if receiver.udpUnix != nil {
		receiver.udpUnix.Close()
	}
	if receiver.tcpNormal != nil {
		receiver.tcpNormal.Close()
	}
}

func (receiver *Client) makeCmdId(cmdId *int64) {
	receiver.lock.Lock()
	defer receiver.lock.Unlock()
	receiver.uniqueId += 1
	if receiver.uniqueId >= math.MaxInt64 {
		receiver.uniqueId = 1
	}
	*cmdId = receiver.uniqueId
}

func (receiver *Client) addWaiter(waiter ResultWaiter) {
	receiver.waiters.Store(waiter.id, waiter)
}

func (receiver *Client) delWaiter(cmdId int64, fn func(*ResultWaiter)) {
	w, ok := receiver.waiters.Load(cmdId)
	if fn != nil {
		if ok {
			v, ok := w.(ResultWaiter)
			if ok {
				fn(&v)
			} else {
				fn(nil)
			}
		}
	}
	if ok {
		receiver.waiters.Delete(cmdId)
	}
}

func (receiver *Client) execWaiter(cmdId int64, fn func(*ResultWaiter)) bool {
	w, ok := receiver.waiters.Load(cmdId)
	if ok {
		v, ok := w.(ResultWaiter)
		if ok {
			fn(&v)
		} else {
			fn(nil)
		}
	}
	return true
}

func (receiver *Client) onKeyValueMsgRet(controller nnet.Controller, _ string, msg codecs.IMData) error {
	mapMsg, ok := msg.(codecs.IMMap)
	if !ok {
		return nil
	} else {
	}

	reader := codecs.CreateMapReader(mapMsg)

	cmdId := reader.IntValueOf(messages.ProtocolKeySerial, 0)
	if cmdId > 0 {
		receiver.delWaiter(cmdId, func(waiter *ResultWaiter) {
			if waiter == nil {
				return
			}
			go func() {
				waiter.ch <- reader.TryReadValue(messages.ProtocolKeyBody)
			}()
		})
	}

	return nil
}

func (receiver *Client) Initialize(addr string, buffWSize int, buffRSize int) error {
	receiver.addr = addr
	//receiver.lookupChan = make(chan interface{}, 10240)
	if strings.Contains(addr, ":") {
		receiver.unixMode = false
		receiver.tcpNormal = nnet.CreateTCPClient(packets.PacketFormatNB, codecs.CodecIMv2)
		receiver.tcpNormal.SetControllerAssociatedObject(receiver)
		receiver.tcpNormal.OnDataDecoded = receiver.onKeyValueMsgRet
		err := receiver.tcpNormal.Connect(addr, 0)
		if err != nil {
			utils.LogInfo("连接 storage => %s 失败. %s", err.Error())
			return err
		}

	} else {
		receiver.unixMode = true
		receiver.udpUnix = nnet.CreateUnixUDPWithFormatAndBufferSize(packets.PacketFormatNB, codecs.CodecIMv2, buffWSize, buffRSize)
		receiver.udpUnix.OnDataDecoded = receiver.onKeyValueMsgRet
		receiver.udpUnix.SetControllerAssociatedObject(receiver)
		myAddr := fmt.Sprintf("/tmp/nbdb_client_%d.sock", os.Getpid())
		err := receiver.udpUnix.Bind(myAddr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (receiver *Client) sendCmdWithRet(cmdData codecs.IMMap) (interface{}, error) {
	defer func() {
	}()

	var cmdId int64 = 0
	receiver.makeCmdId(&cmdId)

	cmdData[messages.ProtocolKeySerial] = cmdId

	recvChan := make(chan interface{})

	tr := time.NewTimer(receiver.timeOut)
	go func() {
		<-tr.C
		receiver.delWaiter(cmdId, func(waiter *ResultWaiter) {
			if waiter != nil {
				go func() {
					recvChan <- nil
				}()
			}
		})
	}()

	waiter := ResultWaiter{id: cmdId, tr: tr, ch: recvChan}
	receiver.addWaiter(waiter)

	if receiver.unixMode {
		cmdData[messages.ProtocolKeyUnixAddr] = receiver.udpUnix.GetBindAddr()
		receiver.udpUnix.SendTo(receiver.addr, cmdData)
	} else {
		receiver.tcpNormal.Send(cmdData)
	}

	ret, ok := <-recvChan
	if ok {
		close(recvChan)
		return ret, nil
	}

	return nil, CmdErrorRet
}

func (receiver *Client) sendCmdWithRetNotTimeout(cmdData codecs.IMMap) (interface{}, error) {
	defer func() {
	}()

	var cmdId int64 = 0
	receiver.makeCmdId(&cmdId)

	cmdData[messages.ProtocolKeySerial] = cmdId

	recvChan := make(chan interface{})

	waiter := ResultWaiter{id: cmdId, tr: nil, ch: recvChan}
	receiver.addWaiter(waiter)

	if receiver.unixMode {
		cmdData[messages.ProtocolKeyUnixAddr] = receiver.udpUnix.GetBindAddr()
		receiver.udpUnix.SendTo(receiver.addr, cmdData)
	} else {
		receiver.tcpNormal.Send(cmdData)
	}

	ret, ok := <-recvChan
	if ok {
		close(recvChan)
		return ret, nil
	}

	return nil, CmdErrorRet
}

func (receiver *Client) sendCmdWithoutRet(cmdData codecs.IMMap) error {
	if receiver.unixMode {
		cmdData[messages.ProtocolKeyUnixAddr] = receiver.udpUnix.GetBindAddr()
		receiver.udpUnix.SendTo(receiver.addr, cmdData)
	} else {
		receiver.tcpNormal.Send(cmdData)
	}
	return nil
}

func (receiver *Client) DBQuery(sql string, args ...interface{}) ([]interface{}, error) {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeDBQuery

	body := make(codecs.IMMap)
	body[messages.ProtocolKeySQL] = sql
	body[messages.ProtocolKeyArgs] = args

	msg[messages.ProtocolKeyBody] = body

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.([]interface{})
		if ok {
			return retV, nil
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("MySQL Query error: %s", sv)
			}
		}
	}
	return nil, err
}

func (receiver *Client) DBExec(sql string, args ...interface{}) (int64, error) {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeDBExec

	body := make(codecs.IMMap)
	body[messages.ProtocolKeySQL] = sql
	body[messages.ProtocolKeyArgs] = args

	msg[messages.ProtocolKeyBody] = body

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		sv, ok := ret.(string)
		if ok {
			utils.LogError("MySQL Exec error: %s", sv)
		} else {
			return codecs.Int64FromInterface(ret), err
		}
	}
	return 0, err
}

func (receiver *Client) DBTransaction(transactions ...Transaction) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeDBTransaction

	body := make(codecs.IMMap)

	sqls := make([]string, len(transactions))
	actions := make([]int, len(transactions))
	args := make([][]interface{}, len(transactions))
	for i, tr := range transactions {
		sqls[i] = tr.sql
		actions[i] = tr.action
		args[i] = tr.args
	}
	body[messages.ProtocolKeySQL] = sqls
	body[messages.ProtocolKeyArgs] = args
	body[messages.ProtocolKeyActions] = actions

	msg[messages.ProtocolKeyBody] = body

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(bool)
		if ok {
			return retV
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("MySQL Transaction error: %s", sv)
			}
		}
	}
	return false
}

func (receiver *Client) RedisOpen(key uint64) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeRedisOpen
	msg[messages.ProtocolKeyKeyForRedis] = key

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(bool)
		if ok {
			return retV
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("Redis error: %s", sv)
			}
		}
	}
	return false
}

func (receiver *Client) RedisClose(key uint64) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeRedisClose
	msg[messages.ProtocolKeyKeyForRedis] = key

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(bool)
		if ok {
			return retV
		}
	}
	return false
}

func (receiver *Client) RedisDo(cmd string, args ...interface{}) interface{} {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeRedisDo

	body := make(codecs.IMMap)
	body[messages.ProtocolKeyCmd] = cmd
	body[messages.ProtocolKeyArgs] = args

	msg[messages.ProtocolKeyBody] = body

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(map[interface{}]interface{})
		if ok {
			m := codecs.CreateMapReader(retV)
			return m.TryReadValue(messages.ProtocolKeyResult)
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("Redis Do error: %s", sv)
			}
		}
	}
	return nil
}

func (receiver *Client) RedisSend(key uint64, cmd string, args ...interface{}) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeRedisSend
	msg[messages.ProtocolKeyKeyForRedis] = key

	body := make(codecs.IMMap)
	body[messages.ProtocolKeyCmd] = cmd
	body[messages.ProtocolKeyArgs] = args

	msg[messages.ProtocolKeyBody] = body

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(bool)
		if ok {
			return retV
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("Redis Send error: %s", sv)
			}
		}
	}
	return false
}

func (receiver *Client) RedisFlush(key uint64) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeRedisFlush
	msg[messages.ProtocolKeyKeyForRedis] = key

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(bool)
		if ok {
			return retV
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("Redis Flush error: %s", sv)
			}
		}
	}
	return false
}

func (receiver *Client) RedisReceive(key uint64) interface{} {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeRedisReceive
	msg[messages.ProtocolKeyKeyForRedis] = key

	ret, err := receiver.sendCmdWithRet(msg)
	if err == nil {
		retV, ok := ret.(map[interface{}]interface{})
		if ok {
			m := codecs.CreateMapReader(retV)
			return m.TryReadValue(messages.ProtocolKeyResult)
		} else {
			sv, ok := ret.(string)
			if ok {
				utils.LogError("Redis Receive error: %s", sv)
			}
		}
	}
	return nil
}

func (receiver *Client) InitLock(key uint64) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeInitLockKey
	msg[messages.ProtocolKeyKeyForLock] = key

	_, err := receiver.sendCmdWithRetNotTimeout(msg)
	return err == nil
}

func (receiver *Client) DisposeLock(key uint64) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeDisposeLockKey
	msg[messages.ProtocolKeyKeyForLock] = key

	_, err := receiver.sendCmdWithRetNotTimeout(msg)
	return err == nil
}

func (receiver *Client) Lock(key uint64) (int64, bool) {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeLockKey
	msg[messages.ProtocolKeyKeyForLock] = key

	iSid, err := receiver.sendCmdWithRetNotTimeout(msg)
	if err == nil {
		return codecs.Int64FromInterface(iSid), true
	}
	return 0, false
}

func (receiver *Client) Unlock(sid int64, key uint64) bool {
	msg := make(codecs.IMMap)
	msg[messages.ProtocolKeyScheme] = messages.ProtocolSchemeS2S
	msg[messages.ProtocolKeyTag] = codecs.IMSlice{messages.ProtocolTagStorage}
	msg[messages.ProtocolKeyType] = messages.ProtocolTypeUnLockKey
	msg[messages.ProtocolKeyKeyForLock] = key
	msg[messages.ProtocolKeySidForLock] = sid

	_, err := receiver.sendCmdWithRetNotTimeout(msg)
	return err == nil
}
