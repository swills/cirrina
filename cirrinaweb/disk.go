package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"

	"cirrina/cirrinactl/rpc"
)

type Disk struct {
	Name        string
	ID          string
	Description string
	Size        string
	Usage       string
}

type DiskHandler struct {
	GetDisk  func(string) (Disk, error)
	GetDisks func() ([]Disk, error)
}

func NewDiskHandler() DiskHandler {
	return DiskHandler{
		GetDisk:  getDisk,
		GetDisks: getDisks,
	}
}

func getDisk(nameOrID string) (Disk, error) {
	var returnDisk Disk

	var diskInfo rpc.DiskInfo

	var err error

	rpc.ServerName = cirrinaServerName
	rpc.ServerPort = cirrinaServerPort
	rpc.ServerTimeout = cirrinaServerTimeout
	rpc.ResetConnTimeout()

	err = rpc.GetConn()
	if err != nil {
		return Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	parsedUUID, err := uuid.Parse(nameOrID)
	if err != nil {
		rpc.ResetConnTimeout()

		returnDisk.ID, err = rpc.DiskNameToID(nameOrID)
		if err != nil {
			return Disk{}, fmt.Errorf("error getting Disk: %w", err)
		}

		returnDisk.Name = nameOrID
	} else {
		returnDisk.ID = parsedUUID.String()

		rpc.ResetConnTimeout()

		returnDisk.Name, err = rpc.GetVMName(parsedUUID.String())
		if err != nil {
			return Disk{}, fmt.Errorf("error getting Disk: %w", err)
		}
	}

	rpc.ResetConnTimeout()

	diskInfo, err = rpc.GetDiskInfo(returnDisk.ID)
	if err != nil {
		return Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	returnDisk.Description = diskInfo.Descr

	var diskSizeUsage rpc.DiskSizeUsage

	diskSizeUsage, err = rpc.GetDiskSizeUsage(returnDisk.ID)
	if err != nil {
		return Disk{}, fmt.Errorf("error getting Disk: %w", err)
	}

	returnDisk.Size = humanize.IBytes(diskSizeUsage.Size)
	returnDisk.Usage = humanize.IBytes(diskSizeUsage.Usage)

	return returnDisk, nil
}

func (d DiskHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	aDisk, err := d.GetDisk(request.PathValue("nameOrID"))
	if err != nil {
		logError(err, request.RemoteAddr)

		serveErrorDisk(writer, request, err)

		return
	}

	Disks, err := d.GetDisks()
	if err != nil {
		t := time.Now()

		_, err = errorLog.WriteString(fmt.Sprintf("[%s] [server:error] [pid %d:tid %d] [client %s] %s\n",
			t.Format("Mon Jan 02 15:04:05.999999999 2006"),
			os.Getpid(),
			0,
			request.RemoteAddr,
			err.Error(),
		))
		if err != nil {
			panic(err)
		}

		http.Error(writer, "failed to retrieve Disks", http.StatusInternalServerError)

		return
	}

	templ.Handler(disk(Disks, aDisk)).ServeHTTP(writer, request)
}
