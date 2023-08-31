package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"fmt"
	"golang.org/x/term"
	"os"
	"time"
)

func UseCom(c cirrina.VMInfoClient, idPtr *string, comNum int) (err error) {
	if *idPtr == "" {
		return errors.New("id not specified")
	}
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
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
		return err
	}

	vmId := &cirrina.VMID{Value: *idPtr}
	req := &cirrina.ComDataRequest{
		Data: &cirrina.ComDataRequest_VmId{
			VmId: vmId,
		},
	}

	err = stream.Send(req)
	if err != nil {
		return err
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
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
			return nil
		default:
			res, _, _, err := GetVMState(idPtr, c, ctx)
			if err != nil {
				_ = stream.CloseSend()
				ctxCancel()
				_ = term.Restore(int(os.Stdin.Fd()), oldState)
				return nil
			}

			if res != "running" {
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
