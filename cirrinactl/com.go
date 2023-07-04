package main

import (
	"cirrina/cirrina"
	"context"
	"fmt"
	"golang.org/x/term"
	"log"
	"os"
	"time"
)

func useCom(c cirrina.VMInfoClient, idPtr *string, comNum int) {
	if *idPtr == "" {
		log.Fatalf("ID not specified")
		return
	}
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	var err error
	var stream cirrina.VMInfo_Com1InteractiveClient

	switch comNum {
	case 1:
		stream, err = c.Com1Interactive(ctx)
	case 2:
		stream, err = c.Com2Interactive(ctx)
	case 3:
		stream, err = c.Com3Interactive(ctx)
	case 4:
		stream, err = c.Com4Interactive(ctx)
	}
	if err != nil {
		log.Fatalf("failed to get stream: %v", err)
		return
	}

	vmId := &cirrina.VMID{Value: *idPtr}
	req := &cirrina.ComDataRequest{
		Data: &cirrina.ComDataRequest_VmId{
			VmId: vmId,
		},
	}

	err = stream.Send(req)
	if err != nil {
		fmt.Printf("streaming com failed: %v\n", err)
	}

	fmt.Print("starting terminal session, press ctrl-\\ to quit\n")
	time.Sleep(1 * time.Second)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(fd int, oldState *term.State) {
		_ = term.Restore(fd, oldState)
	}(int(os.Stdin.Fd()), oldState)

	quitChan := make(chan bool)

	// send
	// FIXME -- cheating a bit here
	go func(stream cirrina.VMInfo_Com1InteractiveClient, oldState *term.State, quitChan chan bool) {
		for {
			select {
			case <-quitChan:
				_ = stream.CloseSend()
				ctxCancel()
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				return
			default:
				b := make([]byte, 1)
				_, err = os.Stdin.Read(b)
				if err != nil {
					if err.Error() != "EOF" {
						fmt.Println(err)
					}
					quitChan <- true
					_ = stream.CloseSend()
					ctxCancel()
					_ = term.Restore(int(os.Stdin.Fd()), oldState)
					return
				}
				if b[0] == 0x1c {
					quitChan <- true
					return
				}
				req := &cirrina.ComDataRequest{
					Data: &cirrina.ComDataRequest_ComInBytes{
						ComInBytes: b,
					},
				}
				err = stream.Send(req)
				if err != nil {
					return
				}
			}
		}
	}(stream, oldState, quitChan)

	// receive
	// FIXME -- cheating a bit here
	go func(stream cirrina.VMInfo_Com1InteractiveClient, oldState *term.State, quitChan chan bool) {

		for {
			select {
			case <-quitChan:
				_ = stream.CloseSend()
				ctxCancel()
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				return
			default:
				out, err := stream.Recv()
				if err != nil {
					_ = stream.CloseSend()
					ctxCancel()
					_ = term.Restore(int(os.Stdin.Fd()), oldState)
					code := err.Error()
					if code == "EOF" {
						fmt.Printf("connection closed\n")
					} else if code != "rpc error: code = Canceled desc = context canceled" {
						fmt.Printf("error receiving from com: %v\n", err)
					}
					return
				}
				fmt.Print(string(out.ComOutBytes))
			}
		}
	}(stream, oldState, quitChan)

	cleared := false
	// monitor
	for {
		select {
		case <-quitChan:
			_ = stream.CloseSend()
			ctxCancel()
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			return
		default:
			res, err := c.GetVMState(ctx, &cirrina.VMID{Value: *idPtr})
			if err != nil {
				_ = stream.CloseSend()
				ctxCancel()
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				return
			}

			if res.Status != cirrina.VmStatus_STATUS_RUNNING {
				_ = stream.CloseSend()
				ctxCancel()
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
			} else {
				if !cleared {
					fmt.Print("\033[H\033[2J")
					cleared = true
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
}
