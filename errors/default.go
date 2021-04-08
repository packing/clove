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

package errors

var ErrorAddress = Errorf("The address is error")
var ErrorDataSentIncomplete = Errorf("The data sent is incomplete")
var ErrorPacketFormatNotReady = Errorf("The packet format is not ready")
var ErrorCodecNotReady = Errorf("The codec is not ready")
var ErrorDecryptFunctionNotBind = Errorf("The packet is encrypted, but the decrypt function is not bind")
var ErrorUncompressFunctionNotBind = Errorf("The packet is compressed, but the uncompress function is not bind")

var ErrorSessionIsNotExists = Errorf("The session is not exists")

var ErrorDataNotEnough = Errorf("The length of the head data is not enough to be decoded")
var ErrorDataTooShort = Errorf("The length of the head data is too short")
var ErrorTypeNotSupported = Errorf("Type is not supported")

var ErrorDataNotReady = Errorf("Data length is not enough")
var ErrorDataNotMatch = Errorf("Cannot match any packet format")
var ErrorDataIsDamage = Errorf("Data length is not match")
var ErrorRemoteReqClose = Errorf("The remote host request close it")
