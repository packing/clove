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

package codecs

import (
	"encoding/json"
)

type DecoderJSONv1 struct {
}

type EncoderJSONv1 struct {
}


func (receiver DecoderJSONv1) Decode(raw []byte) (error, IMData, []byte){
	dst := make(map[string] interface{})
	err := json.Unmarshal(raw, &dst)
	if err != nil && err.Error() == "json: cannot unmarshal array into Go value of type map[string]interface {}" {
		dst := make([]interface{},0)
		err = json.Unmarshal(raw, &dst)
	}
	return err, dst, []byte("")
}

func (receiver EncoderJSONv1) Encode(raw *IMData) (error, []byte){
	bs, err := json.Marshal(raw)
	return err, bs
}

var codecJSONv1 = Codec{Protocol:ProtocolJSON, Version:1, Decoder: DecoderJSONv1{}, Encoder: EncoderJSONv1{}}
var CodecJSONv1 = &codecJSONv1