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
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/internal/auth"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/internal/model"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/internal/store"
)

type Config struct {
	BookingStore *store.BookingStore
	Authorizer   *auth.Authorizer
}

const (
	objectWildcard         = "*"
	getBookingByUUIDAction = "getBookingByUUID"
	getBookingByIDAction   = "getBookingByID"
	createBookingAction    = "createBooking"
	updateBookingAction    = "updateBooking"
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
		grpc.StreamInterceptor(grpcmdwr.ChainStreamServer(
			grpcctxtags.StreamServerInterceptor(),
			grpczap.StreamServerInterceptor(logger, zapOpts...),
			grpcauth.StreamServerInterceptor(authenticate),
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

func (s *grpcServer) GetBookingByUUID(ctx context.Context,
	req *api.GetByUUIDBookingRequest) (*api.GetBookingResponse, error) {

	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		getBookingByUUIDAction,
	); err != nil {
		return nil, err
	}
	b, err := s.BookingStore.GetByUUID(req.UUID)
	if err != nil {
		return nil, api.NewErrBookingNotFoundForUUID(req.UUID).ErrBooking
	}
	return &api.GetBookingResponse{Booking: b.ProtoBooking()}, nil
}

func (s *grpcServer) GetBookingByID(ctx context.Context,
	req *api.GetByIDBookingRequest) (*api.GetBookingResponse, error) {

	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		getBookingByIDAction,
	); err != nil {
		return nil, err
	}
	b, err := s.BookingStore.GetByID(req.ID)
	if err != nil {
		return nil, api.NewErrBookingNotFoundForID(req.ID).ErrBooking
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

func (s *grpcServer) UpdateBooking(ctx context.Context,
	req *api.UpdateBookingRequest) (*api.UpdateBookingResponse, error) {

	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		updateBookingAction,
	); err != nil {
		return nil, err
	}
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
