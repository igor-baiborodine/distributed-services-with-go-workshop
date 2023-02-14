package server

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServerSideServiceDiscovery/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServerSideServiceDiscovery/internal/auth"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServerSideServiceDiscovery/internal/config"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServerSideServiceDiscovery/internal/store"
)

func TestServer(t *testing.T) {
	tests := []struct {
		desc string
		fn   func(t *testing.T, rootClient api.BookingServiceClient,
			nobodyClient api.BookingServiceClient, cfg *Config)
	}{
		{"create and get by UUID booking succeeds", testCreateGetByUUID},
		{"create and get by ID booking succeeds", testCreateGetByID},
		{"create and update booking succeeds", testCreateUpdate},
		{"create and get booking stream succeeds",
			testCreateGetBookingStream},
		{"get non-existing booking fails", testGetNonExisting},
		{"unauthorized fails", testUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			rootClient, nobodyClient, cfg, teardown := setupTest(t, nil)
			defer teardown()
			test.fn(t, rootClient, nobodyClient, cfg)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (
	rootClient api.BookingServiceClient,
	nobodyClient api.BookingServiceClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	newClient := func(crtPath, keyPath string) (
		*grpc.ClientConn,
		api.BookingServiceClient,
		[]grpc.DialOption,
	) {
		tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   config.CAFile,
			Server:   false,
		})
		require.NoError(t, err)
		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := api.NewBookingServiceClient(conn)
		return conn, client, opts
	}

	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ = newClient(
		config.RootClientCertFile,
		config.RootClientKeyFile,
	)

	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(
		config.NobodyClientCertFile,
		config.NobodyClientKeyFile,
	)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	bs, err := store.NewBookingStore()
	require.NoError(t, err)

	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)
	cfg = &Config{
		BookingStore: bs,
		Authorizer:   authorizer,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return rootClient, nobodyClient, cfg, func() {
		server.Stop()
		rootConn.Close()
		nobodyConn.Close()
		l.Close()
	}
}

func testCreateGetByUUID(t *testing.T, rootClient api.BookingServiceClient,
	nobodyClient api.BookingServiceClient, config *Config) {

	ctx := context.Background()
	want, err := createBooking(rootClient, ctx)
	require.NoError(t, err)

	got, err := rootClient.GetBookingByUUID(ctx,
		&api.GetByUUIDBookingRequest{UUID: want.UUID})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
}

func testCreateGetByID(t *testing.T, rootClient api.BookingServiceClient,
	nobodyClient api.BookingServiceClient, config *Config) {

	ctx := context.Background()
	want, err := createBooking(rootClient, ctx)
	require.NoError(t, err)

	got, err := rootClient.GetBookingByID(ctx, &api.GetByIDBookingRequest{ID: 1})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
}

func testCreateUpdate(t *testing.T, rootClient api.BookingServiceClient,
	nobodyClient api.BookingServiceClient, config *Config) {

	ctx := context.Background()
	want, err := createBooking(rootClient, ctx)
	require.NoError(t, err)
	want.StartDate = "2023-02-15"
	want.EndDate = "2023-02-18"

	got, err := rootClient.UpdateBooking(ctx,
		&api.UpdateBookingRequest{Booking: want})
	require.NoError(t, err)
	want.ID++
	assertBooking(t, want, got.Booking)
}

func testGetNonExisting(t *testing.T, rootClient api.BookingServiceClient,
	nobodyClient api.BookingServiceClient, config *Config) {

	ctx := context.Background()
	u := uuid.New().String()
	_, err := rootClient.GetBookingByUUID(ctx, &api.GetByUUIDBookingRequest{UUID: u})
	require.Errorf(t, err, "no booking found for UUID: %s", u)
}

func testCreateGetBookingStream(t *testing.T,
	rootClient api.BookingServiceClient, nobodyClient api.BookingServiceClient,
	config *Config) {

	ctx := context.Background()
	booking1 := createBookingProto(1, "john.smith@dot.com", "John Smith",
		"2023-01-20", "2023-01-23")
	booking2 := createBookingProto(2, "jack.jones@dot.com", "Jack Jones",
		"2023-01-27", "2023-01-30")
	booking3 := createBookingProto(3, "robert.brown@dot.com", "Robert Brown",
		"2023-02-11", "2023-02-15")

	var bookings = []*api.Booking{booking1, booking2, booking3}

	{
		stream, err := rootClient.CreateBookingStream(ctx)
		require.NoError(t, err)

		for i, b := range bookings {
			err = stream.Send(&api.CreateBookingRequest{Booking: b})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)

			if res.Booking.ID != uint64(i+1) {
				t.Fatalf("got ID: %d, want: %d", res.Booking.ID, i+1)
			}
		}
	}

	{
		stream, err := rootClient.GetBookingStream(
			ctx, &api.GetByIDBookingRequest{ID: 1})
		require.NoError(t, err)

		for _, b := range bookings {
			res, err := stream.Recv()
			require.NoError(t, err)
			assertBooking(t, b, res.Booking)
		}
	}
}

func testUnauthorized(t *testing.T, rootClient api.BookingServiceClient,
	nobodyClient api.BookingServiceClient, _ *Config) {

	ctx := context.Background()
	got, err := nobodyClient.GetBookingByUUID(ctx,
		&api.GetByUUIDBookingRequest{UUID: uuid.New().String()})

	require.Nilf(t, got, "get booking response should be nil")
	require.Error(t, err)

	s := status.Convert(err)
	assert.Equal(t, s.Code(), codes.PermissionDenied)
	assert.Equal(t, s.Message(), "nobody not permitted to getBookingByUUID to *")
}

func createBooking(client api.BookingServiceClient,
	ctx context.Context) (*api.Booking, error) {

	b := createBookingProto(1, "john.smith@dot.com", "John Smith",
		"2023-01-20", "2023-01-23")
	_, err := client.CreateBooking(ctx,
		&api.CreateBookingRequest{Booking: b})
	return b, err
}

func createBookingProto(id uint64, email string, fullname string,
	startDate string, endDate string) *api.Booking {

	return &api.Booking{
		ID:        id,
		UUID:      uuid.New().String(),
		Email:     email,
		FullName:  fullname,
		StartDate: startDate,
		EndDate:   endDate,
		Active:    true,
	}
}

func assertBooking(t *testing.T, want *api.Booking, got *api.Booking) {

	require.Equal(t, want.ID, got.ID)
	require.Equal(t, want.UUID, got.UUID)
	require.Equal(t, want.Email, got.Email)
	require.Equal(t, want.FullName, got.FullName)
	require.Equal(t, want.StartDate, got.StartDate)
	require.Equal(t, want.EndDate, got.EndDate)
	require.Equal(t, want.Active, got.Active)
}
