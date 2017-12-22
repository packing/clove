package utils

import (
	"log"
	"fmt"
	"os"
	"time"
	"io"
)

const (
	LogLevelVerbose = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var LogLevelTags = []string {"verbose", "info", "warn", "error"}

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
		for i := 0; i < 4; i++ {
			fn := fmt.Sprintf("%s-%s-%s.log", logPrefix, LogLevelTags[i], dt)
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
		}
	}

	return nil
}

func getLogIO(level int) (error, io.Writer) {
	dt := time.Now().Format("2006-01-02")
	fn := fmt.Sprintf("%s-%s-%s.log", logPrefix, LogLevelTags[level], dt)
	pf, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND, 0)
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
		err, iowriter := getLogIO(level)
		if err != nil {
			return
		}
		logger = log.New(iowriter, LogLevelTags[level], log.Lshortfile | log.Ltime)
		logger.Println(s)
	} else {
		log.SetFlags(log.Ltime)
		log.SetPrefix(fmt.Sprintf("[%s]", LogLevelTags[level]))
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

