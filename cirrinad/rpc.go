package main

import (
	"cirrina/cirrina"
	"cirrina/cirrinad/config"
	"log/slog"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type server struct {
	cirrina.UnimplementedVMInfoServer
}

func isOptionPassed(reflect protoreflect.Message, name string) bool {
	field := reflect.Descriptor().Fields().ByName(protoreflect.Name(name))
	return reflect.Has(field)
}

func rpcServer() {
	listenAddress := config.Config.Network.Grpc.Ip + ":" + strconv.Itoa(int(config.Config.Network.Grpc.Port))
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		slog.Error("failed to listen for rpc", "listenAddress", listenAddress, "err", err)
	}
	s := grpc.NewServer()
	// Register reflection service on gRPC server.
	reflection.Register(s)
	cirrina.RegisterVMInfoServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve rpc", "err", err)
	}
}
