package main

import (
	"time"
	"net"


	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"

	pb "github.com/yjp211/go-replica/rpc"

	"github.com/yjp211/go-replica/quic"
)

type CreateRpcClient func(addr string) (pb.RpcGreeterClient, error);


func CreateTcpRpcClient(addr string) (pb.RpcGreeterClient, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return pb.NewRpcGreeterClient(conn), nil
}

func CreateQuicRpcClient(addr string) (pb.RpcGreeterClient, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	//封装连接函数
	opts = append(opts, grpc.WithDialer(func(s string, duration time.Duration) (net.Conn, error) {
		return quic.Dial(s, duration)
	}));


	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return pb.NewRpcGreeterClient(conn), nil
}
