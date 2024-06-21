package main

import (
	"log/slog"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
)

type server struct {
	cirrina.UnimplementedVMInfoServer
}

func rpcServer() {
	listenAddress := config.Config.Network.Grpc.IP + ":" + strconv.Itoa(int(config.Config.Network.Grpc.Port))
	lis, err := net.Listen("tcp", listenAddress)

	if err != nil {
		slog.Error("failed to listen for rpc", "listenAddress", listenAddress, "err", err)
	}

	s := grpc.NewServer()
	// Register reflection service on gRPC server.
	reflection.Register(s)
	cirrina.RegisterVMInfoServer(s, &server{})

	err = s.Serve(lis)
	if err != nil {
		slog.Error("failed to serve rpc", "err", err)
	}
}
