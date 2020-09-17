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
	"log"
	"fmt"
	"os"
	"time"
	"io"
	"runtime"
)

const (
	LogLevelVerbose = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var LogLevelTags = []string {"V", "I", "W", "E"}

var loggerVerbose log.Logger
var loggerInfo log.Logger
var loggerWarn log.Logger
var loggerError log.Logger

var logLevel = LogLevelVerbose
var logPrefix string

func LogInit(level int, prefix string) (error) {
	logLevel = level
	logPrefix = prefix
	if len(logPrefix) > 0 {
		dt := time.Now().Format("2006-01-02")
		//for i := 0; i < 4; i++ {
			fn := fmt.Sprintf("%s-%s.log", logPrefix, dt)
			_, err := os.Stat(fn)
			if os.IsExist(err) {
				for j := 1; j < 100; j ++ {
					sfn := fmt.Sprintf("%s.%d", fn, j)
					_, err := os.Stat(sfn)
					if os.IsExist(err) {
						continue
					}
					os.Rename(fn, sfn)
				}
			}
			pf, err := os.Create(fn)
			pf.Close()
			if err != nil {
				return err
			}
		//}
	}

	return nil
}

func getLogIO() (error, io.Writer) {
	dt := time.Now().Format("2006-01-02")
	fn := fmt.Sprintf("%s-%s.log", logPrefix, dt)
	pf, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		defer pf.Close()
		return err, nil
	}
	return nil, pf
}

func logAndLevel(level int, format string, v ...interface{}) {
	if level < logLevel {
		return
	}
	s := fmt.Sprintf(format, v...)
	var logger *log.Logger = nil
	if len(logPrefix) > 0 {
		err, iowriter := getLogIO()
		if err != nil {
			return
		}
		logger = log.New(iowriter, fmt.Sprintf("[%s]", LogLevelTags[level]), log.Ltime)
		logger.Println(s)
	} else {
		log.SetFlags(log.Ltime)
		log.SetPrefix(fmt.Sprintf("[%s]", LogLevelTags[level]))
		log.Println(s)
	}
}

func LogRaw(s string) {
	var logger *log.Logger = nil
	if len(logPrefix) > 0 {
		err, iowriter := getLogIO()
		if err != nil {
			return
		}
		logger = log.New(iowriter, "", 0)
		logger.Println(s)
	} else {
		log.Println(s)
	}
}

func LogVerbose(format string, v ...interface{}) {
	logAndLevel(LogLevelVerbose, format, v...)
}

func LogInfo(format string, v ...interface{}) {
	logAndLevel(LogLevelInfo, format, v...)
}

func LogWarn(format string, v ...interface{}) {
	logAndLevel(LogLevelWarn, format, v...)
}

func LogError(format string, v ...interface{}) {
	logAndLevel(LogLevelError, format, v...)
}

func LogPanic(ierr interface{}) {
	if ierr == nil {
		return
	}
	err, ok := ierr.(error)
	if ok && err != nil {
		var st = func(all bool) string {
			// Reserve 1K buffer at first
			buf := make([]byte, 512)

			for {
				size := runtime.Stack(buf, all)
				// The size of the buffer may be not enough to hold the stacktrace,
				// so double the buffer size
				if size == len(buf) {
					buf = make([]byte, len(buf)<<1)
					continue
				}
				break
			}

			return string(buf)
		}
		LogError("panic:", err ,"\nstack:" + st(false))
	}
}