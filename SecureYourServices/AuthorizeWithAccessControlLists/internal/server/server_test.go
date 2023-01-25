package server

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/internal/auth"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/internal/config"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/internal/store"
)

func TestServer(t *testing.T) {
	tests := []struct {
		desc string
		fn   func(t *testing.T, rootClient api.BookingServiceClient,
			nobodyClient api.BookingServiceClient, cfg *Config)
	}{
		{"create/get booking succeeds", testCreateGet},
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

func testCreateGet(t *testing.T, rootClient api.BookingServiceClient,
	_ api.BookingServiceClient, _ *Config) {
	ctx := context.Background()
	want := &api.Booking{
		UUID:      uuid.New().String(),
		Email:     "john.smith@dot.com",
		FullName:  "John Smith",
		StartDate: "2023-01-20",
		EndDate:   "2023-01-23",
	}

	_, err := rootClient.CreateBooking(ctx,
		&api.CreateBookingRequest{Booking: want})
	require.NoError(t, err)

	got, err := rootClient.GetBooking(
		ctx, &api.GetBookingRequest{Uuid: want.UUID})
	require.NoError(t, err)
	require.Equal(t, want.UUID, got.Booking.UUID)
	require.Equal(t, want.Email, got.Booking.Email)
	require.Equal(t, want.FullName, got.Booking.FullName)
	require.Equal(t, want.StartDate, got.Booking.StartDate)
	require.Equal(t, want.EndDate, got.Booking.EndDate)
	require.Equal(t, true, got.Booking.Active)
}

func testGetNonExisting(t *testing.T, rootClient api.BookingServiceClient,
	_ api.BookingServiceClient, _ *Config) {
	ctx := context.Background()
	u := uuid.New().String()

	_, err := rootClient.GetBooking(ctx, &api.GetBookingRequest{Uuid: u})
	require.Error(t, err)
	assert.Contains(t, err.Error(),
		fmt.Sprintf("no booking found for UUID: %s", u))
}

func testUnauthorized(t *testing.T, _ api.BookingServiceClient,
	nobodyClient api.BookingServiceClient, _ *Config) {
	ctx := context.Background()
	got, err := nobodyClient.GetBooking(ctx,
		&api.GetBookingRequest{Uuid: uuid.New().String()})

	require.Nilf(t, got, "get booking response should be nil")
	require.Error(t, err)

	s := status.Convert(err)
	assert.Equal(t, s.Code(), codes.PermissionDenied)
	assert.Equal(t, s.Message(), "nobody not permitted to getBooking to *")
}
