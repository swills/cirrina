package rpc

import (
	"cirrina/cirrina"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strconv"
	"time"
)

var ServerName string
var ServerPort uint16
var ServerTimeout uint64
var ServerTimeoutDur time.Duration

func SetupConn() (*grpc.ClientConn, cirrina.VMInfoClient, context.Context, context.CancelFunc, error) {
	serverAddr := ServerName + ":" + strconv.FormatInt(int64(ServerPort), 10)
	serverTimeoutDur := time.Second * time.Duration(ServerTimeout)

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	c := cirrina.NewVMInfoClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), serverTimeoutDur)
	return conn, c, ctx, cancel, nil
}

func SetupConnNoTimeoutNoContext() (*grpc.ClientConn, cirrina.VMInfoClient, error) {
	serverAddr := ServerName + ":" + strconv.FormatInt(int64(ServerPort), 10)

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	c := cirrina.NewVMInfoClient(conn)

	return conn, c, nil
}
