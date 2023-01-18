package server

import (
	"context"
	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/model"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/store"
	"google.golang.org/grpc"
)

type Config struct {
	BookingStore store.BookingStore
}

type grpcServer struct {
	*Config
	*api.UnimplementedBookingServiceServer
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		config,
		&api.UnimplementedBookingServiceServer{},
	}
	return srv, nil
}

func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterBookingServiceServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) GetBooking(_ context.Context, req *api.GetBookingRequest) (
	*api.GetBookingResponse, error) {
	b, err := s.BookingStore.GetByUUID(req.Uuid)
	if err != nil {
		return nil, api.ErrBookingNotFound{UUID: req.GetUuid()}
	}
	return &api.GetBookingResponse{Booking: b.ProtoBooking()}, nil
}

type BookingStore interface {
	GetByUUID(uuid string) (model.Booking, error)
}
