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

package messages

const (
	ProtocolSchemeC2S 		= 0x01
	ProtocolSchemeS2C 		= 0x02
	ProtocolSchemeS2S 		= 0x03

	ProtocolKeyScheme 		= 0x09
	ProtocolKeyType 		= 0x10
	ProtocolKeyTag 			= 0x11
	ProtocolKeySessionId 	= 0x12
	ProtocolKeySync 		= 0x13
	ProtocolKeyBody 		= 0x14
	ProtocolKeyErrorCode 	= 0x15
	ProtocolKeySerial 		= 0x16
	ProtocolKeyId 			= 0x17
    ProtocolKeyValue 	    = 0x18
    ProtocolKeyCpu 	        = 0x19
    ProtocolKeyMem 	        = 0x20
    ProtocolKeyGoroutine 	= 0x21
	ProtocolKeyUnixAddr 	= 0x22
	ProtocolKeyTcpAddr 		= 0x23
	ProtocolKeyHost 		= 0x24

	ProtocolTypeDeliver 	= 0x01
	ProtocolTypeHeart 		= 0x02
	ProtocolTypeClientLeave = 0x03
	ProtocolTypeSlaveHello 	= 0x04
	ProtocolTypeAdapterHello= 0x05
	ProtocolTypeGatewayHello= 0x06
	ProtocolTypeKillClient	= 0x07
	ProtocolTypeAdapters	= 0x08

    ProtocolTagMaster 		= 0x0
	ProtocolTagSlave 		= 0x01
    ProtocolTagAdapter 		= 0x02
	ProtocolTagClient 		= 0x03

	ProtocolErrorCodeOK 	= 0
)