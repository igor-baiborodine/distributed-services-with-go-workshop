package server

import (
	"context"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/internal/config"
	"google.golang.org/grpc/credentials"
	"net"
	"testing"

	"github.com/google/uuid"
	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/internal/store"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api.BookingServiceClient,
		cfg *Config,
	){
		"create/get booking succeeds":    testCreateGet,
		"get non-existing booking fails": testGetNonExisting,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, cfg, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, cfg)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (
	client api.BookingServiceClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile: config.CAFile,
	})
	require.NoError(t, err)

	clientCreds := credentials.NewTLS(clientTLSConfig)
	cc, err := grpc.Dial(
		l.Addr().String(),
		grpc.WithTransportCredentials(clientCreds),
	)
	require.NoError(t, err)

	client = api.NewBookingServiceClient(cc)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	bs, err := store.NewBookingStore()
	require.NoError(t, err)

	cfg = &Config{
		BookingStore: bs,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client = api.NewBookingServiceClient(cc)

	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
	}
}

func testCreateGet(t *testing.T, client api.BookingServiceClient, _ *Config) {
	ctx := context.Background()
	want := &api.Booking{
		UUID:      uuid.New().String(),
		Email:     "john.smith@dot.com",
		FullName:  "John Smith",
		StartDate: "2023-01-20",
		EndDate:   "2023-01-23",
	}

	_, err := client.CreateBooking(ctx, &api.CreateBookingRequest{Booking: want})
	require.NoError(t, err)

	got, err := client.GetBooking(ctx, &api.GetBookingRequest{Uuid: want.UUID})
	require.NoError(t, err)
	require.Equal(t, want.UUID, got.Booking.UUID)
	require.Equal(t, want.Email, got.Booking.Email)
	require.Equal(t, want.FullName, got.Booking.FullName)
	require.Equal(t, want.StartDate, got.Booking.StartDate)
	require.Equal(t, want.EndDate, got.Booking.EndDate)
	require.Equal(t, true, got.Booking.Active)
}

func testGetNonExisting(t *testing.T, client api.BookingServiceClient, _ *Config) {
	ctx := context.Background()
	u := uuid.New().String()

	_, err := client.GetBooking(ctx, &api.GetBookingRequest{Uuid: u})
	require.Errorf(t, err, "no booking found for UUID: %s", u)
}
