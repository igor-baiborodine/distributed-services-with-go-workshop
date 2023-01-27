package server

import (
	"context"
	"time"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	grpcmdwr "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/internal/model"
)

type Config struct {
	BookingStore BookingStore
	Authorizer   Authorizer
}

const (
	objectWildcard      = "*"
	createBookingAction = "createBooking"
	getBookingAction    = "getBooking"
)

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

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) (
	*grpc.Server,
	error,
) {
	logger := zap.L().Named("server")
	zapOpts := []grpczap.Option{
		grpczap.WithDurationField(
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
		grpc.UnaryInterceptor(
			grpcmdwr.ChainUnaryServer(
				grpcctxtags.UnaryServerInterceptor(),
				grpczap.UnaryServerInterceptor(logger, zapOpts...),
				grpcauth.UnaryServerInterceptor(authenticate),
			)),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	)
	gsrv := grpc.NewServer(grpcOpts...)
	srv, err := newgrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterBookingServiceServer(gsrv, srv)
	return gsrv, nil
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
	b, err := s.BookingStore.GetByUUID(req.Uuid)
	if err != nil {
		return nil, api.ErrBookingNotFound{UUID: req.GetUuid()}
	}
	return &api.GetBookingResponse{Booking: b.ProtoBooking()}, nil
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
