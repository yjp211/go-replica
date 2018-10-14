package main

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"

	"github.com/yjp211/go-replica/log"
	"github.com/yjp211/go-replica/quic"
	pb "github.com/yjp211/go-replica/rpc"
	"net"
)

type Server struct {
}

func (s *Server) Ping(ctx context.Context, req *pb.Nothing) (*pb.Nothing, error) {
	return req, nil
}

func (s *Server) Echo(ctx context.Context, req *pb.Msg) (*pb.Msg, error) {
	return req, nil
}

func (s *Server) Replica(stream pb.RpcGreeter_ReplicaServer) error {
	for {
		ctx := stream.Context()

		switch val, err := stream.Recv(); {
		case err == nil:
			ProducerRecord(val)

		case err == io.EOF:
			time.Sleep(1 * time.Second) // provoke race condition
			select {
			case <-ctx.Done():
				log.Error("Context is already DONE!")
			default:
				log.Info("Context is OK")
			}
			return nil
		default:
			return err
		}
	}
}

func StartGRpcTcpServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("failed to listen tcp: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterRpcGreeterServer(s, &Server{})
	go s.Serve(lis)
	log.Info("grpc listening tcp: %v\n", lis.Addr())
}


func StartGRpcQuicServer(port int) {
	lis, err := quic.ListenAddr(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("failed to listen quic: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterRpcGreeterServer(s, &Server{})
	go s.Serve(lis)
	log.Info("grpc listening quic: %v\n", lis.Addr())
}
