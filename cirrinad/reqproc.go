package main

import (
	"time"

	"cirrina/cirrinad/requests"
)

func processRequests() {
	for {
		rs := requests.GetUnStarted()
		if rs.ID != "" {
			rs.Start()
			switch rs.Type {
			case requests.VMSTART:
				go startVM(&rs)
			case requests.VMSTOP:
				go stopVM(&rs)
			case requests.VMDELETE:
				go deleteVM(&rs)
			case requests.NICCLONE:
				go nicClone(&rs)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}
