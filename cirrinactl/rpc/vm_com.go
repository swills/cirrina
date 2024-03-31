package rpc

import (
	"cirrina/cirrina"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
	"google.golang.org/grpc/status"
)

func UseCom(id string, comNum int) error {
	var err error

	if id == "" {
		return errors.New("id not specified")
	}
	bgCtx, cancel := context.WithCancel(context.Background())
	var stream cirrina.VMInfo_Com1InteractiveClient

	switch comNum {
	case 1:
		stream, err = serverClient.Com1Interactive(bgCtx)
	case 2:
		stream, err = serverClient.Com2Interactive(bgCtx)
	case 3:
		stream, err = serverClient.Com3Interactive(bgCtx)
	case 4:
		stream, err = serverClient.Com4Interactive(bgCtx)
	}
	if err != nil {
		cancel()
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
		cancel()
		return err
	}

	// save term state and set up restore when done
	var oldState *term.State
	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		cancel()
		return err
	}
	defer func(fd int, oldState *term.State) {
		_ = stream.CloseSend()
		_ = term.Restore(fd, oldState)
	}(int(os.Stdin.Fd()), oldState)

	// clear screen
	fmt.Print("\033[H\033[2J")

	// send
	go func(stream cirrina.VMInfo_Com1InteractiveClient) {
		for {
			select {
			case <-bgCtx.Done():
				return
			default:
				b := make([]byte, 1)
				_, err = os.Stdin.Read(b)
				if err != nil {
					cancel()
					return
				}
				if b[0] == 0x1c { // == FS ("File Separator") control character -- ctrl-\ -- see ascii.7
					cancel()
					return
				}
				req := &cirrina.ComDataRequest{
					Data: &cirrina.ComDataRequest_ComInBytes{
						ComInBytes: b,
					},
				}
				err = stream.Send(req)
				if err != nil {
					cancel()
					return
				}
			}
		}
	}(stream)

	// receive
	go func(stream cirrina.VMInfo_Com1InteractiveClient) {
		for {
			select {
			case <-bgCtx.Done():
				return
			default:
				var out *cirrina.ComDataResponse
				out, err = stream.Recv()
				if err != nil {
					cancel()
					return
				}
				fmt.Print(string(out.ComOutBytes))
			}
		}
	}(stream)

	// monitor
	for {
		select {
		case <-bgCtx.Done():
			return nil
		default:
			var res string
			ResetConnTimeout()
			res, _, _, err = GetVMState(id)
			if err != nil {
				cancel()
				return nil
			}

			if res != "running" && res != "stopping" {
				cancel()
				return nil
			}
			time.Sleep(1 * time.Second)
		}
	}
}
