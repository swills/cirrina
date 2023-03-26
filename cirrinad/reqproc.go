package main

import (
	"time"
)

func processRequests() {
	for {
		rs := getUnStartedReq()
		if rs.ID != "" {
			startReq(rs)
			switch rs.Type {
			case START:
				go startVM(&rs)
			case STOP:
				go stopVM(&rs)
			case DELETE:
				go deleteVM(&rs)
			}

		}
		time.Sleep(500 * time.Millisecond)
	}
}
