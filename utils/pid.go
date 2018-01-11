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
	"fmt"
	"os"
	"strings"
	"strconv"
)

func GeneratePID(pidFile string) {
	pf, err := os.OpenFile(pidFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		LogError("Write pidfile error!", err)
		return
	}

	pf.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	pf.Close()
}

func RemovePID(pidFile string) {
	err, pids := ReadPIDs(pidFile)
	if err != nil {
		LogError("Remove from pidfile error!", err)
		return
	}

	pf, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	if err != nil {
		LogError("Remove from pidfile error!", err)
		return
	}

	for _,pid := range pids {
		if pid == os.Getpid() {
			continue
		}
		pf.WriteString(fmt.Sprintf("%d\n", pid))
	}
	pf.Close()
}

func ReadPIDs(pidFile string) (error, []int) {
	pf, err := os.Open(pidFile)
	if err != nil {
		LogError("Remove from pidfile error!", err)
		return err, nil
	}

	bs := make([]byte, 10240)
	n, err := pf.Read(bs)
	pf.Close()
	if err != nil {
		LogError("Remove from pidfile error!", err)
		return err, nil
	}
	ctx := string(bs[:n])
	pids := strings.Split(ctx, "\n")
	intpids := make([]int, len(pids))
	i := 0
	for _, v := range pids {
		nv, err := strconv.Atoi(v)
		if err == nil {
			intpids[i] = nv
			i ++
		}
	}
	return nil, intpids[:i]
}