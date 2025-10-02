package server

import (
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	s   *grpc.Server
	lis net.Listener
}

func (g *GRPCServer) Serve() error    { return g.s.Serve(g.lis) }
func (g *GRPCServer) Shutdown() error { g.s.GracefulStop(); return nil }
