package main

import (
	"time"

	"cirrina/cirrinad/requests"
)

func processRequests() {
	for {
		request := requests.GetUnStarted()
		if request.ID != "" {
			request.Start()

			switch request.Type {
			case requests.VMSTART:
				go startVM(&request)
			case requests.VMSTOP:
				go stopVM(&request)
			case requests.VMDELETE:
				go deleteVM(&request)
			case requests.NICCLONE:
				go nicClone(&request)
			case requests.DISKWIPE:
				go diskWipe(&request)
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}
