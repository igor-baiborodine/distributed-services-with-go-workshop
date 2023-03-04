package server

import (
	"context"
	"time"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/api/v1"
)

type Config struct {
	BookingLog BookingLog
	Authorizer Authorizer
}

const (
	objectWildcard      = "*"
	produceAction       = "produce"
	consumeAction       = "consume"
	getBookingAction    = "getBooking"
	createBookingAction = "createBooking"
	updateBookingAction = "updateBooking"
)

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

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) (
	*grpc.Server, error) {

	logger := zap.L().Named("server")
	zapOpts := []grpc_zap.Option{
		grpc_zap.WithDurationField(
			func(duration time.Duration) zapcore.Field {
				return zap.Int64(
					"grpc.time_ns",
					duration.Nanoseconds(),
				)
			},
		),
	}
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	err := view.Register(ocgrpc.DefaultServerViews...)
	if err != nil {
		return nil, err
	}

	grpcOpts = append(grpcOpts,
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_ctxtags.StreamServerInterceptor(),
				grpc_zap.StreamServerInterceptor(logger, zapOpts...),
				grpc_auth.StreamServerInterceptor(authenticate),
			)), grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(logger, zapOpts...),
			grpc_auth.UnaryServerInterceptor(authenticate),
		)),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	)
	gsrv := grpc.NewServer(grpcOpts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (
	*api.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}

	offset, err := s.BookingLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (
	*api.ConsumeResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		consumeAction,
	); err != nil {
		return nil, err
	}

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
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		getBookingAction,
	); err != nil {
		return nil, err
	}

	b, err := s.BookingLog.ReadBooking(req.Uuid)
	if err != nil {
		return nil, api.NewErrNotFoundForUUID(req.Uuid)
	}
	return &api.GetBookingResponse{Booking: b}, nil
}

func (s *grpcServer) CreateBooking(ctx context.Context,
	req *api.CreateBookingRequest) (*api.CreateBookingResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		createBookingAction,
	); err != nil {
		return nil, err
	}

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
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		updateBookingAction,
	); err != nil {
		return nil, err
	}

	b, err := s.BookingLog.ReadBooking(req.Booking.Uuid)
	if err != nil {
		return nil, api.NewErrNotFoundForUUID(req.Booking.Uuid)
	}
	req.Booking.Active = true
	req.Booking.CreatedAt = b.CreatedAt
	req.Booking.UpdatedAt = timestamppb.New(time.Now())

	_, err = s.BookingLog.AppendBooking(req.Booking)
	if err != nil {
		return nil, api.NewErrUpdateBooking(req.Booking)
	}
	return &api.UpdateBookingResponse{Booking: req.Booking}, nil
}

type BookingLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
	AppendBooking(booking *api.Booking) (uint64, error)
	ReadBooking(uuid string) (*api.Booking, error)
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

func authenticate(ctx context.Context) (context.Context, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if p.AuthInfo == nil {
		return ctx, status.New(
			codes.Unauthenticated,
			"no transport security being used",
		).Err()
	}

	tlsInfo := p.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
