package server

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	metricPkg "metrics-and-alerting/pkg/metric"
	pb "metrics-and-alerting/proto"
	"net"
)

type GRPCServer struct {
	*grpc.Server
	net.Listener
}

type MetricsServiceRPC struct {
	pb.UnimplementedMetricsServer
	m *MetricsManager
}

func NewGRPCServer(addr string, m *MetricsManager) (*GRPCServer, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	g := GRPCServer{
		Server:   grpc.NewServer(),
		Listener: listen,
	}

	service := &MetricsServiceRPC{
		m: m,
	}

	pb.RegisterMetricsServer(g.Server, service)
	return &g, nil
}

func (g *GRPCServer) Start() {
	go func() {
		if err := g.Server.Serve(g.Listener); err != nil {
			fmt.Printf("failed run gRPC server: %v\n", err)
		}
	}()
}

func (serv *MetricsServiceRPC) UpsertGauge(ctx context.Context, in *pb.UpsertGaugeRequest) (*pb.UpsertResponse, error) {

	var response pb.UpsertResponse

	metric, err := metricPkg.CreateMetric(
		metricPkg.GaugeType,
		in.Id,
		metricPkg.WithValueFloat(in.Value),
	)

	if err == nil {
		err = serv.m.Upsert(metric)
	}

	if err != nil {
		response.Error = err.Error()
	}

	return &response, nil
}

func (serv *MetricsServiceRPC) UpsertCounter(ctx context.Context, in *pb.UpsertCounterRequest) (*pb.UpsertResponse, error) {

	var response pb.UpsertResponse

	metric, err := metricPkg.CreateMetric(
		metricPkg.CounterType,
		in.Id,
		metricPkg.WithValueInt(in.Delta),
	)

	if err == nil {
		err = serv.m.Upsert(metric)
	}

	if err != nil {
		response.Error = err.Error()
	}

	return &response, nil
}
