package server

import (
	"context"

	"google.golang.org/grpc"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateClientWithMutualTLSAuthentication/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateClientWithMutualTLSAuthentication/internal/model"
)

type Config struct {
	BookingStore BookingStore
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

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (
	*grpc.Server,
	error,
) {
	gsrv := grpc.NewServer(opts...)
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

func (s *grpcServer) CreateBooking(_ context.Context, req *api.CreateBookingRequest) (
	*api.CreateBookingResponse, error) {

	b := model.Booking{
		UUID:      req.GetBooking().UUID,
		Email:     req.GetBooking().Email,
		FullName:  req.GetBooking().FullName,
		StartDate: req.GetBooking().StartDate,
		EndDate:   req.GetBooking().EndDate,
		Active:    true,
	}
	err := s.BookingStore.Create(b)
	if err != nil {
		return nil, api.ErrCreateBooking{Booking: req.GetBooking()}
	}
	return &api.CreateBookingResponse{Booking: b.ProtoBooking()}, nil
}

type BookingStore interface {
	GetByUUID(uuid string) (model.Booking, error)
	Create(b model.Booking) error
}
