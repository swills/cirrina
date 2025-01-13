package util

import (
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/grpc/status"

	"cirrina/cirrinactl/rpc"
)

var (
	serverName           = "localhost"
	serverPort    uint16 = 50051
	serverTimeout uint64 = 5
)

func InitRPCConn() error {
	var err error

	rpc.ServerName = serverName
	rpc.ServerPort = serverPort
	rpc.ServerTimeout = serverTimeout
	rpc.ResetConnTimeout()

	err = rpc.GetConn()
	if err != nil {
		return fmt.Errorf("error initializing RPC connection: %w", err)
	}

	return nil
}

func InitRPC(serverNameI string, serverPortI string, serverTimeoutI string) {
	var err error

	if serverNameI != "" {
		serverName = serverNameI
	}

	if serverPortI != "" {
		var cirrinaServerPortTemp uint64

		cirrinaServerPortTemp, err = strconv.ParseUint(serverPortI, 10, 16)
		if err == nil {
			serverPort = uint16(cirrinaServerPortTemp)
		}
	}

	if serverTimeoutI != "" {
		var serverTimeoutTemp uint64

		serverTimeoutTemp, err = strconv.ParseUint(serverTimeoutI, 10, 64)
		if err == nil {
			serverTimeout = serverTimeoutTemp
		}
	}
}

func GetServerName() string {
	return serverName
}

func GetErrDesc(err error) string {
	var errMessage string

	var lastErr error

	currentErr := err
	for currentErr != nil {
		lastErr = currentErr
		currentErr = errors.Unwrap(currentErr)
	}

	s, ok := status.FromError(lastErr)

	if ok {
		errMessage = s.Message()
	} else {
		errMessage = err.Error()
	}

	return errMessage
}
