package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"fmt"
	"golang.org/x/term"
	"google.golang.org/grpc/status"
	"os"
	"time"
)

func UseCom(id string, comNum int) error {
	var err error

	if id == "" {
		return errors.New("id not specified")
	}
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	var stream cirrina.VMInfo_Com1InteractiveClient

	switch comNum {
	case 1:
		stream, err = serverClient.Com1Interactive(ctx)
	case 2:
		stream, err = serverClient.Com2Interactive(ctx)
	case 3:
		stream, err = serverClient.Com3Interactive(ctx)
	case 4:
		stream, err = serverClient.Com4Interactive(ctx)
	}
	if err != nil {
		return errors.New(status.Convert(err).Message())
	}

	vmId := &cirrina.VMID{Value: id}
	req := &cirrina.ComDataRequest{
		Data: &cirrina.ComDataRequest_VmId{
			VmId: vmId,
		},
	}

	err = stream.Send(req)
	if err != nil {
		return err
	}
	var oldState *term.State
	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
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
				if b[0] == 0x1c { // == FS ("File Separator") control character -- ctrl-\ -- see ascii.7
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
				var out *cirrina.ComDataResponse
				out, err = stream.Recv()
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
	// prevent timeouts
	defaultServerContext = context.Background()
	// monitor
	for {
		select {
		case <-quitChan:
			_ = stream.CloseSend()
			ctxCancel()
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			return nil
		default:
			var res string
			res, _, _, err = GetVMState(id)
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
