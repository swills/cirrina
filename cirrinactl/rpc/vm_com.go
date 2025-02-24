package rpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	"cirrina/cirrina"
)

func UseCom(vmID string, comNum int) error {
	var err error

	var stream cirrina.VMInfo_Com1InteractiveClient

	if vmID == "" {
		return errVMEmptyID
	}

	defaultServerContext, defaultCancelFunc = context.WithCancel(context.Background())

	switch comNum {
	case 1:
		stream, err = serverClient.Com1Interactive(defaultServerContext)
	case 2:
		stream, err = serverClient.Com2Interactive(defaultServerContext)
	case 3:
		stream, err = serverClient.Com3Interactive(defaultServerContext)
	case 4:
		stream, err = serverClient.Com4Interactive(defaultServerContext)
	default:
		defaultCancelFunc()

		return ErrInvalidComNum
	}

	if err != nil {
		defaultCancelFunc()

		return fmt.Errorf("unable to use com: %w", err)
	}

	// setup stream
	err = comStreamSetup(vmID, stream)
	if err != nil {
		defaultCancelFunc()

		return err
	}

	// save term state
	oldState, err := comTermSetup()
	if err != nil {
		defaultCancelFunc()

		return err
	}

	defer func(stream cirrina.VMInfo_Com1InteractiveClient) {
		comStreamCleanup(stream)
	}(stream)
	defer func(oldState *term.State) {
		comTermCleanup(oldState)
	}(oldState)

	// send
	go comSend(stream)

	// receive
	go comReceive(stream)

	// monitor that the VM is still up
	return comMonitorVM(vmID)
}

func comTermCleanup(oldState *term.State) {
	_ = term.Restore(int(os.Stdin.Fd()), oldState)
}

func comStreamCleanup(stream cirrina.VMInfo_Com1InteractiveClient) {
	_ = stream.CloseSend()
}

func comMonitorVM(vmID string) error {
	var err error

	for {
		select {
		case <-defaultServerContext.Done():
			defaultCancelFunc()

			return nil
		default:
			var res string

			ResetConnTimeout()

			res, _, _, err = GetVMState(vmID)
			if err != nil {
				defaultCancelFunc()

				return fmt.Errorf("unable to use com: %w", err)
			}

			if res != "running" && res != "stopping" {
				defaultCancelFunc()

				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func comTermSetup() (*term.State, error) {
	var err error

	var oldState *term.State

	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("unable to use com: %w", err)
	}

	// clear screen
	fmt.Print("\033[H\033[2J")

	return oldState, nil
}

func comStreamSetup(vmID string, stream cirrina.VMInfo_Com1InteractiveClient) error {
	var err error

	req := &cirrina.ComDataRequest{
		Data: &cirrina.ComDataRequest_VmId{
			VmId: &cirrina.VMID{Value: vmID},
		},
	}

	err = stream.Send(req)
	if err != nil {
		return fmt.Errorf("unable to use com: %w", err)
	}

	return nil
}

// comSend reads data from the local terminal and sends it to the remote serial port
func comSend(stream cirrina.VMInfo_Com1InteractiveClient) {
	var err error

	var req *cirrina.ComDataRequest

	bytesBuffer := make([]byte, 1)

	for {
		select {
		case <-defaultServerContext.Done():
			return
		default:
			_, err = os.Stdin.Read(bytesBuffer)
			if err != nil {
				defaultCancelFunc()

				return
			}

			if bytesBuffer[0] == 0x1c { // == FS ("File Separator") control character -- ctrl-\ -- see ascii.7
				defaultCancelFunc()

				return
			}

			req = &cirrina.ComDataRequest{
				Data: &cirrina.ComDataRequest_ComInBytes{
					ComInBytes: bytesBuffer,
				},
			}

			err = stream.Send(req)
			if err != nil {
				defaultCancelFunc()

				return
			}
		}
	}
}

// comReceive receives data from the remote serial port and outputs it to the local terminal
func comReceive(stream cirrina.VMInfo_Com1InteractiveClient) {
	var err error

	var out *cirrina.ComDataResponse

	for {
		select {
		case <-defaultServerContext.Done():
			return
		default:
			out, err = stream.Recv()
			if err != nil {
				defaultCancelFunc()

				return
			}

			fmt.Print(string(out.GetComOutBytes()))
		}
	}
}
