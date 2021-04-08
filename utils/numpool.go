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

package utils

import (
	"math/rand"
	"time"
)

func randomIntSlice(vs []int64) []int64 {
	rand.Seed(time.Now().Unix())

	if len(vs) <= 0 {
		return vs
	}

	newVs := make([]int64, len(vs))
	copy(newVs, vs)

	for i := len(newVs) - 1; i > 0; i-- {
		num := rand.Intn(i + 1)
		newVs[i], newVs[num] = newVs[num], newVs[i]
	}

	return newVs
}

func CreateNumPool(s, e int64, random bool) []int64 {
	c := e - s + 1
	l := make([]int64, c)
	for i := s; i <= e; i++ {
		l[i-s] = i
	}

	if random {
		return randomIntSlice(l)
	}
	return l
}

func Shuffle(s []interface{}) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(s) > 0 {
		n := len(s)
		i := r.Intn(n)
		s[n-1], s[i] = s[i], s[n-1]
		s = s[:n-1]
	}
}
