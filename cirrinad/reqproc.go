package main

import (
	"cirrina/cirrinad/requests"
	"time"
)

func processRequests() {
	for {
		rs := requests.GetUnStartedReq()
		if rs.ID != "" {
			requests.StartReq(rs)
			switch rs.Type {
			case requests.START:
				go startVM(&rs)
			case requests.STOP:
				go stopVM(&rs)
			case requests.DELETE:
				go deleteVM(&rs)
			}

		}
		time.Sleep(500 * time.Millisecond)
	}
}
