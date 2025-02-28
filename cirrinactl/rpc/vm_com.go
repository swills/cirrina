package rpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	"cirrina/cirrina"
)

var oldState *term.State

func UseCom(vmID string, comNum int) error {
	var err error

	var stream cirrina.VMInfo_Com1InteractiveClient

	if vmID == "" {
		return errVMEmptyID
	}

	comCtx, comCancel := context.WithCancel(context.Background())

	switch comNum {
	case 1:
		stream, err = serverClient.Com1Interactive(comCtx)
	case 2:
		stream, err = serverClient.Com2Interactive(comCtx)
	case 3:
		stream, err = serverClient.Com3Interactive(comCtx)
	case 4:
		stream, err = serverClient.Com4Interactive(comCtx)
	default:
		comCancel()

		return ErrInvalidComNum
	}

	if err != nil {
		comCancel()

		return fmt.Errorf("unable to use com: %w", err)
	}

	// setup stream
	err = comStreamSetup(vmID, stream)
	if err != nil {
		comCancel()

		return err
	}

	// save term state
	oldState, err = comTermSetup()
	if err != nil {
		comCancel()

		return err
	}

	defer func(stream cirrina.VMInfo_Com1InteractiveClient) {
		comStreamCleanup(stream)
	}(stream)

	// send
	go comSend(comCtx, comCancel, stream)

	// receive
	go comReceive(comCtx, comCancel, stream)

	// monitor that the VM is still up
	return comMonitorVM(comCtx, comCancel, vmID)
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

func comStreamCleanup(stream cirrina.VMInfo_Com1InteractiveClient) {
	_ = stream.CloseSend()
}

func comTermSetup() (*term.State, error) {
	var err error

	oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("unable to use com: %w", err)
	}

	// clear screen
	fmt.Print("\033[H\033[2J")

	return oldState, nil
}

func comTermCleanup(message string) {
	_ = term.Restore(int(os.Stdin.Fd()), oldState)

	if message != "" {
		fmt.Printf("\n" + message + "\n")
	}
}

func comMonitorVM(comCtx context.Context, comCancel context.CancelFunc, vmID string) error {
	var err error

	for {
		select {
		case <-comCtx.Done():
			return nil
		default:
			var res string

			res, _, _, err = GetVMState(comCtx, vmID)
			if err != nil {
				select {
				case <-comCtx.Done():
					return nil
				default:
					comCancel()
					comTermCleanup("unable to monitor com: %s" + err.Error())

					return fmt.Errorf("unable to monitor com: %w", err)
				}
			}

			if res != "running" && res != "stopping" && res != "starting" {
				comCancel()
				comTermCleanup("VM shutdown")

				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}

// comSend reads data from the local terminal and sends it to the remote serial port
func comSend(comCtx context.Context, comCancel context.CancelFunc, stream cirrina.VMInfo_Com1InteractiveClient) {
	var err error

	var req *cirrina.ComDataRequest

	bytesBuffer := make([]byte, 1)

	for {
		select {
		case <-comCtx.Done():
			return
		default:
			_, err = os.Stdin.Read(bytesBuffer)
			if err != nil {
				comCancel()
				comTermCleanup("failed reading stdin: " + err.Error())

				return
			}

			if bytesBuffer[0] == 0x1c { // == FS ("File Separator") control character -- ctrl-\ -- see ascii.7
				comCancel()
				comTermCleanup("disconnected")

				return
			}

			req = &cirrina.ComDataRequest{
				Data: &cirrina.ComDataRequest_ComInBytes{
					ComInBytes: bytesBuffer,
				},
			}

			err = stream.Send(req)
			if err != nil {
				select {
				case <-comCtx.Done():
					return
				default:
					comCancel()
					comTermCleanup("failed sending to com: " + err.Error())
				}

				return
			}
		}
	}
}

// comReceive receives data from the remote serial port and outputs it to the local terminal
func comReceive(comCtx context.Context, comCancel context.CancelFunc, stream cirrina.VMInfo_Com1InteractiveClient) {
	var err error

	var out *cirrina.ComDataResponse

	for {
		select {
		case <-comCtx.Done():
			return
		default:
			out, err = stream.Recv()
			if err != nil {
				select {
				case <-comCtx.Done():
					return
				default:
					comCancel()
					comTermCleanup("failed receiving from com: " + err.Error())
				}

				return
			}

			fmt.Print(string(out.GetComOutBytes()))
		}
	}
}
