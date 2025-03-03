package main

import (
	"log/slog"
	"net"
	"strconv"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"cirrina/cirrina"
	"cirrina/cirrinad/config"
)

type server struct {
	cirrina.UnimplementedVMInfoServer
}

// ensure we meet the interface
var _ cirrina.VMInfoServer = &server{}

func rpcServer() {
	listenAddress := config.Config.Network.Grpc.IP + ":" +
		strconv.FormatUint(cast.ToUint64(config.Config.Network.Grpc.Port), 10)
	lis, err := net.Listen("tcp", listenAddress)

	if err != nil {
		slog.Error("failed to listen for rpc", "listenAddress", listenAddress, "err", err)
	}

	var opts []grpc.ServerOption

	// opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
	//	Time:                  time.Duration(config.Config.Network.Grpc.Timeout) * time.Second,
	//	Timeout:               time.Duration(config.Config.Network.Grpc.Timeout) * time.Second,
	//	MaxConnectionAgeGrace: time.Duration(config.Config.Network.Grpc.Timeout) * time.Second,
	//	MaxConnectionIdle:     time.Duration(config.Config.Network.Grpc.Timeout) * time.Second,
	//	MaxConnectionAge:      time.Duration(config.Config.Network.Grpc.Timeout) * time.Second,
	// }))
	//
	// connTimeout := grpc.ConnectionTimeout(time.Duration(config.Config.Network.Grpc.Timeout) * time.Second)
	//
	// opts = append(opts, connTimeout)

	opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
		Time: time.Duration(config.Config.Network.Grpc.Timeout) * time.Second,
	}))

	var srvMetrics *grpcprom.ServerMetrics

	if config.Config.Metrics.Enabled {
		srvMetrics = grpcprom.NewServerMetrics(
			grpcprom.WithServerHandlingTimeHistogram(
				grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
			),
		)
		prometheus.DefaultRegisterer.MustRegister(srvMetrics)
		streamInterceptor := srvMetrics.StreamServerInterceptor()
		unaryInterceptor := srvMetrics.UnaryServerInterceptor()
		opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptor))

		opts = append(opts, grpc.ChainStreamInterceptor(streamInterceptor))
	}

	grpcSrv := grpc.NewServer(opts...)

	if config.Config.Metrics.Enabled {
		if srvMetrics != nil {
			srvMetrics.InitializeMetrics(grpcSrv)
		}
	}

	// Register reflection service on gRPC server.
	reflection.Register(grpcSrv)
	cirrina.RegisterVMInfoServer(grpcSrv, &server{})

	err = grpcSrv.Serve(lis)
	if err != nil {
		slog.Error("failed to serve rpc", "err", err)
	}
}
