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

package caches

import (
    "github.com/packing/nbpy/nnet"
    "github.com/packing/nbpy/codecs"
    "fmt"
    "os"
    "github.com/packing/nbpy/errors"
    "strings"
    "github.com/packing/nbpy/packets"
    "reflect"
)

const (
    KeyValueCmdGet = 0xF000
    KeyValueCmdSet = 0xF001
    KeyValueCmdDel = 0xF002
    KeyValueCmdRet = 0xF003

    KeyValueFieldSid  = 0x13
    KeyValueFieldType = 0x10
    KeyValueFieldFrom = 0x11
    KeyValueFieldReq  = 0x12

    KeyValueFieldKey   = 0x89
    KeyValueFieldValue = 0x90
    KeyValueFieldCode  = 0x91
)

var KeyValueErrorRet = errors.Errorf("Keyvalue cacheServer return fail")

func convertToInt64(v interface{}, def int64) (int64) {
    if v == nil {
        return def
    }
    kind := reflect.TypeOf(v).Kind()
    switch kind {
    case reflect.Int:
        fallthrough
    case reflect.Int8:
        fallthrough
    case reflect.Int16:
        fallthrough
    case reflect.Int32:
        fallthrough
    case reflect.Int64:
        return reflect.ValueOf(v).Int()
    case reflect.Uint:
        fallthrough
    case reflect.Uint8:
        fallthrough
    case reflect.Uint16:
        fallthrough
    case reflect.Uint32:
        fallthrough
    case reflect.Uint64:
        return int64(reflect.ValueOf(v).Uint())
    default:
        return def
    }
}

type KeyValueCache struct {
    udpUnix    *nnet.UnixUDP
    tcpNormal  *nnet.TCPClient
    addr       string
    keepAlive  bool // only tcp mode
    lookupChan chan interface{}
}

func CreateKeyValueCache(addr string, keepAlive bool) (*KeyValueCache) {
    kv := new(KeyValueCache)
    kv.Initialize(addr, keepAlive)
    return kv
}

func onKeyValueMsgRet(controller nnet.Controller, _ string, msg codecs.IMData) error {
    ao := controller.GetAssociatedObject()
    if ao == nil {
        return nil
    }

    kv, ok := ao.(*KeyValueCache)
    if !ok {
        return nil
    }

    if !kv.keepAlive && strings.Contains(kv.addr, ":") {
        controller.Close()
    }

    mapMsg, ok := msg.(codecs.IMMap)
    if !ok {
        return nil
    } else {
    }

    reader := codecs.CreateMapReader(mapMsg)

    tp := reader.IntValueOf(KeyValueFieldReq, 0)
    if tp == 0 {
        return nil
    }

    key := reader.StrValueOf(KeyValueFieldKey, "")
    if key == "" {
        return nil
    }

    code := reader.IntValueOf(KeyValueFieldCode, -1)

    d := reader.TryReadValue(KeyValueFieldValue)

    switch tp {
    case KeyValueCmdGet:
        if kv != nil {
            kv.CmdRet(d)
        }

    case KeyValueCmdSet:
        if kv != nil {
            kv.CmdRet(code == 0)
        }
    case KeyValueCmdDel:
        if kv != nil {
            kv.CmdRet(code == 0)
        }
    }
    return nil
}

func (receiver *KeyValueCache) Initialize(addr string, keepAlive bool) (error) {
    receiver.addr = addr
    receiver.keepAlive = keepAlive
    receiver.lookupChan = make(chan interface{}, 10240)
    if strings.Contains(addr, ":") {
        receiver.tcpNormal = nnet.CreateTCPClient(packets.PacketFormatNB, codecs.CodecIMv2)
        receiver.tcpNormal.SetControllerAssociatedObject(receiver)
        receiver.tcpNormal.OnDataDecoded = onKeyValueMsgRet
        if receiver.keepAlive {
            err := receiver.tcpNormal.Connect(addr, 0)
            if err != nil {
                return err
            }
        } else {
            return nil
        }
    } else {
        receiver.udpUnix = nnet.CreateUnixUDPWithFormat(packets.PacketFormatNB, codecs.CodecIMv2)
        receiver.udpUnix.OnDataDecoded = onKeyValueMsgRet
        receiver.udpUnix.SetControllerAssociatedObject(receiver)
        myAddr := fmt.Sprintf("/tmp/nbkv_client_%d.sock", os.Getpid())
        err := receiver.udpUnix.Bind(myAddr)
        if err != nil {
            return err
        }
    }
    return nil
}

func (receiver *KeyValueCache) CmdRet(ret interface{}) {
    go func() {
        receiver.lookupChan <- ret
    }()
}

func (receiver *KeyValueCache) execCmd(cmd int, key string, value interface{}) (interface{}, error) {
    req := make(codecs.IMMap)
    req[KeyValueFieldType] = cmd
    req[KeyValueFieldKey] = key
    if value != nil {
        req[KeyValueFieldValue] = value
    }
    req[KeyValueFieldSid] = fmt.Sprintf("nbkv_sid_%d_%d", os.Getpid(), nnet.NewSessionID())
    if receiver.udpUnix != nil {
        myAddr := fmt.Sprintf("/tmp/nbkv_client_%d.sock", os.Getpid())
        req[KeyValueFieldFrom] = myAddr
        receiver.udpUnix.SendTo(receiver.addr, req)
    } else {
        if !receiver.keepAlive {
            err := receiver.tcpNormal.Connect(receiver.addr, 0)
            if err != nil {
                return nil, err
            }
        }
        receiver.tcpNormal.Send(req)
    }
    ret, ok := <-receiver.lookupChan
    if ok {
        return ret, nil
    } else {
        return nil, KeyValueErrorRet
    }
}

func (receiver *KeyValueCache) GetValue(key string) (interface{}, error) {
    return receiver.execCmd(KeyValueCmdGet, key, "")
}

func (receiver *KeyValueCache) GetString(key string, def string) (string) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        rstr, ok := r.(string)
        if ok {
            return rstr
        } else {
            return def
        }
    }
    return def
}

func (receiver *KeyValueCache) GetInt(key string, def int) (int) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return int(i)
    }
    return def
}

func (receiver *KeyValueCache) GetInt8(key string, def int8) (int8) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return int8(i)
    }
    return def
}

func (receiver *KeyValueCache) GetInt16(key string, def int16) (int16) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return int16(i)
    }
    return def
}

func (receiver *KeyValueCache) GetInt32(key string, def int32) (int32) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return int32(i)
    }
    return def
}

func (receiver *KeyValueCache) GetInt64(key string, def int64) (int64) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return i
    }
    return def
}

func (receiver *KeyValueCache) GetUint(key string, def uint) (uint) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return uint(i)
    }
    return def
}

func (receiver *KeyValueCache) GetUint8(key string, def uint8) (uint8) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return uint8(i)
    }
    return def
}

func (receiver *KeyValueCache) GetUint16(key string, def uint16) (uint16) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return uint16(i)
    }
    return def
}

func (receiver *KeyValueCache) GetUint32(key string, def uint32) (uint32) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return uint32(i)
    }
    return def
}

func (receiver *KeyValueCache) GetUint64(key string, def uint64) (uint64) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        i := convertToInt64(r, int64(def))
        return uint64(i)
    }
    return def
}

func (receiver *KeyValueCache) GetFloat32(key string, def float32) (float32) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        ri, ok := r.(float32)
        if ok {
            return ri
        } else {
            return def
        }
    }
    return def
}

func (receiver *KeyValueCache) GetFloat64(key string, def float64) (float64) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        ri, ok := r.(float64)
        if ok {
            return ri
        } else {
            return def
        }
    }
    return def
}

func (receiver *KeyValueCache) GetBool(key string) (bool) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        ri, ok := r.(bool)
        if ok {
            return ri
        } else {
            return false
        }
    }
    return false
}

func (receiver *KeyValueCache) GetMap(key string) (codecs.IMMap) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        ri, ok := r.(codecs.IMMap)
        if ok {
            return ri
        } else {
            return nil
        }
    }
    return nil
}

func (receiver *KeyValueCache) GetSlice(key string) (codecs.IMSlice) {
    r, err := receiver.execCmd(KeyValueCmdGet, key, nil)
    if err == nil {
        ri, ok := r.(codecs.IMSlice)
        if ok {
            return ri
        } else {
            return nil
        }
    }
    return nil
}

func (receiver *KeyValueCache) SetValue(key string, value interface{}) (error) {
    _, err := receiver.execCmd(KeyValueCmdSet, key, value)
    return err
}

func (receiver *KeyValueCache) RemoveValue(key string) (error) {
    _, err := receiver.execCmd(KeyValueCmdDel, key, nil)
    return err
}
