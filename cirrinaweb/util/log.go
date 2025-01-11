package util

import (
	"fmt"
	"os"
	"time"
)

var accessLog *os.File
var errorLog *os.File

func LogError(err error, remoteAddr string) {
	t := time.Now()

	_, err = errorLog.WriteString(fmt.Sprintf("[%s] [server:error] [pid %d:tid %d] [client %s] %s\n",
		t.Format("Mon Jan 02 15:04:05.999999999 2006"),
		os.Getpid(),
		0,
		remoteAddr,
		err.Error(),
	))
	if err != nil {
		panic(err)
	}
}

func SetAccessLog(accessLogFile string) {
	var err error

	if accessLogFile == "" {
		accessLog = os.Stdout
	} else {
		accessLog, err = os.OpenFile(accessLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			accessLog = os.Stdout
		}
	}
}

func SetErrorLog(errorLogFile string) {
	var err error

	if errorLogFile == "" {
		errorLog = os.Stderr
	} else {
		errorLog, err = os.OpenFile(errorLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			errorLog = os.Stderr
		}
	}
}

func GetAccessLog() *os.File {
	return accessLog
}
