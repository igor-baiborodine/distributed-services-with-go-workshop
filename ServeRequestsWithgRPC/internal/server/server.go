package server

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/api/v1"
)

type Config struct {
	BookingLog BookingLog
}

type grpcServer struct {
	*Config
	*api.UnimplementedLogServer
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		config,
		&api.UnimplementedLogServer{},
	}
	return srv, nil
}

func NewGRPCServer(config *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (
	*api.ProduceResponse, error) {
	offset, err := s.BookingLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (
	*api.ConsumeResponse, error) {
	record, err := s.BookingLog.Read(req.Offset)
	if err != nil {
		return nil, api.NewErrNotFoundForOffset(req.Offset)
	}
	return &api.ConsumeResponse{Record: record}, nil
}

func (s *grpcServer) ProduceStream(
	stream api.Log_ProduceStreamServer,
) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		res, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *grpcServer) ConsumeStream(
	req *api.ConsumeRequest,
	stream api.Log_ConsumeStreamServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrNotFoundForOffset:
				continue
			default:
				return err
			}
			if err = stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}

func (s *grpcServer) GetBooking(ctx context.Context,
	req *api.GetBookingRequest) (*api.GetBookingResponse, error) {
	b, err := s.BookingLog.ReadBooking(req.Uuid)
	if err != nil {
		return nil, api.NewErrNotFoundForUUID(req.Uuid)
	}
	return &api.GetBookingResponse{Booking: b}, nil
}

func (s *grpcServer) CreateBooking(ctx context.Context,
	req *api.CreateBookingRequest) (*api.CreateBookingResponse, error) {
	req.Booking.Active = true
	req.Booking.CreatedAt = timestamppb.New(time.Now())
	req.Booking.UpdatedAt = nil

	_, err := s.BookingLog.AppendBooking(req.Booking)
	if err != nil {
		return nil, api.NewErrCreateBooking(req.Booking)
	}
	return &api.CreateBookingResponse{Booking: req.Booking}, nil
}

func (s *grpcServer) UpdateBooking(ctx context.Context,
	req *api.UpdateBookingRequest) (*api.UpdateBookingResponse, error) {

	b, err := s.BookingLog.ReadBooking(req.Booking.Uuid)
	if err != nil {
		return nil, api.NewErrNotFoundForUUID(req.Booking.Uuid)
	}
	req.Booking.Active = true
	req.Booking.CreatedAt = b.CreatedAt
	req.Booking.UpdatedAt = timestamppb.New(time.Now())

	_, err = s.BookingLog.AppendBooking(req.Booking)
	if err != nil {
		return nil, api.NewErrCreateBooking(req.Booking)
	}
	return &api.UpdateBookingResponse{Booking: req.Booking}, nil
}

type BookingLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
	AppendBooking(booking *api.Booking) (uint64, error)
	ReadBooking(uuid string) (*api.Booking, error)
}
