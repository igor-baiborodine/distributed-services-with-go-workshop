package server

import (
	"context"
	"time"

	"google.golang.org/grpc"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/model"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/store"
)

type Config struct {
	BookingStore *store.BookingStore
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

func (s *grpcServer) CreateBooking(_ context.Context, req *api.CreateBookingRequest) (
	*api.CreateBookingResponse, error) {

	b := model.Booking{
		UUID:      req.GetBooking().UUID,
		Email:     req.GetBooking().Email,
		FullName:  req.GetBooking().FullName,
		StartDate: req.GetBooking().StartDate,
		EndDate:   req.GetBooking().EndDate,
		Active:    true,
		CreatedAt: time.Now(),
	}
	err := s.BookingStore.Create(b)
	if err != nil {
		return nil, api.ErrCreateBooking{Booking: req.GetBooking()}
	}
	return &api.CreateBookingResponse{Booking: b.ProtoBooking()}, nil
}

func (s *grpcServer) UpdateBooking(_ context.Context,
	req *api.UpdateBookingRequest) (
	*api.UpdateBookingResponse, error) {

	eb, err := s.BookingStore.GetByUUID(req.GetBooking().UUID)
	if err != nil {
		return nil, api.ErrUpdateBooking{Booking: req.GetBooking()}
	}

	b := model.Booking{
		UUID:      req.GetBooking().UUID,
		Email:     req.GetBooking().Email,
		FullName:  req.GetBooking().FullName,
		StartDate: req.GetBooking().StartDate,
		EndDate:   req.GetBooking().EndDate,
		Active:    true,
		CreatedAt: eb.CreatedAt,
		UpdatedAt: time.Now(),
	}
	err = s.BookingStore.Update(b)
	if err != nil {
		return nil, api.ErrUpdateBooking{Booking: req.GetBooking()}
	}
	return &api.UpdateBookingResponse{Booking: b.ProtoBooking()}, nil
}

type BookingStore interface {
	GetByUUID(uuid string) (model.Booking, error)
	Create(b model.Booking) error
	Update(b model.Booking) error
}
