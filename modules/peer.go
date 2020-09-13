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

package modules

import "github.com/packing/nbpy/nnet"

const (
	TunnelDirect 	= 1
	TunnelUPNP		= 2
	TunnelStun		= 4
	TunnelAll		= TunnelDirect | TunnelUPNP | TunnelStun
)

type PeerInfo struct {
	AddrWAN string
	PortWAN string
	PortLAN string
	TunnelType int
}

type Peer struct {
	io *net.UDP
	local PeerInfo
	trustPeers []PeerInfo
	peers map[net.SessionID] PeerInfo
}

func (receiver *Peer) Init() (error) {
	return receiver.InitWithTunnelType(TunnelAll)
}

func (receiver *Peer) InitWithTunnelType(tunnels int) (error) {
	if tunnels & TunnelDirect == TunnelDirect {
		//尝试直接连接可能性

	}
	return nil
}

func (receiver *Peer) tryDirectConnect() (error) {
	return nil
}