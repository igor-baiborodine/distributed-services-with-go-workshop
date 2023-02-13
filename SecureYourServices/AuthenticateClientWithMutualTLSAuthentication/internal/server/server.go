package server

import (
	"context"
	"time"

	"google.golang.org/grpc"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateClientWithMutualTLSAuthentication/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateClientWithMutualTLSAuthentication/internal/model"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateClientWithMutualTLSAuthentication/internal/store"
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

func NewGRPCServer(config *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterBookingServiceServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) GetBookingByUUID(_ context.Context,
	req *api.GetByUUIDBookingRequest) (*api.GetBookingResponse, error) {

	b, err := s.BookingStore.GetByUUID(req.UUID)
	if err != nil {
		return nil, api.NewErrBookingNotFoundForUUID(req.UUID).ErrBooking
	}
	return &api.GetBookingResponse{Booking: b.ProtoBooking()}, nil
}

func (s *grpcServer) GetBookingByID(_ context.Context,
	req *api.GetByIDBookingRequest) (*api.GetBookingResponse, error) {

	b, err := s.BookingStore.GetByID(req.ID)
	if err != nil {
		return nil, api.NewErrBookingNotFoundForID(req.ID).ErrBooking
	}
	return &api.GetBookingResponse{Booking: b.ProtoBooking()}, nil
}

func (s *grpcServer) CreateBooking(_ context.Context,
	req *api.CreateBookingRequest) (*api.CreateBookingResponse, error) {

	b := model.Booking{
		UUID:      req.Booking.UUID,
		Email:     req.Booking.Email,
		FullName:  req.Booking.FullName,
		StartDate: req.Booking.StartDate,
		EndDate:   req.Booking.EndDate,
		Active:    true,
		CreatedAt: time.Now(),
	}
	cb, err := s.BookingStore.Create(b)
	if err != nil {
		return nil, api.NewErrCreateBooking(req.Booking).ErrBooking
	}
	return &api.CreateBookingResponse{Booking: cb.ProtoBooking()}, nil
}

func (s *grpcServer) UpdateBooking(_ context.Context,
	req *api.UpdateBookingRequest) (*api.UpdateBookingResponse, error) {

	eb, err := s.BookingStore.GetByUUID(req.Booking.UUID)
	if err != nil {
		return nil, api.NewErrUpdateBooking(req.Booking).ErrBooking
	}

	b := model.Booking{
		UUID:      req.Booking.UUID,
		Email:     req.Booking.Email,
		FullName:  req.Booking.FullName,
		StartDate: req.Booking.StartDate,
		EndDate:   req.Booking.EndDate,
		Active:    true,
		CreatedAt: eb.CreatedAt,
		UpdatedAt: time.Now(),
	}
	ub, err := s.BookingStore.Update(b)
	if err != nil {
		return nil, api.NewErrUpdateBooking(req.Booking).ErrBooking
	}
	return &api.UpdateBookingResponse{Booking: ub.ProtoBooking()}, nil
}

func (s *grpcServer) CreateBookingStream(
	stream api.BookingService_CreateBookingStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.CreateBooking(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *grpcServer) GetBookingStream(req *api.GetByIDBookingRequest,
	stream api.BookingService_GetBookingStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.GetBookingByID(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrBooking:
				continue
			default:
				return err
			}
			if err = stream.Send(res); err != nil {
				return err
			}
			req.ID++
		}
	}
}

type BookingStore interface {
	GetByUUID(UUID string) (model.Booking, error)
	GetByID(ID uint64) (model.Booking, error)
	Create(b model.Booking) (model.Booking, error)
	Update(b model.Booking) (model.Booking, error)
}
