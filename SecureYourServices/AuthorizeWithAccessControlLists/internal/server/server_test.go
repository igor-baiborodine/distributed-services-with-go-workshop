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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/internal/config"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/internal/store"
)

func TestServer(t *testing.T) {
	tests := []struct {
		desc string
		fn   func(t *testing.T, client api.BookingServiceClient, cfg *Config)
		tls  bool
	}{
		{"create/get booking succeeds", testCreateGet, true},
		{"get non-existing booking fails", testGetNonExisting, true},
		{"insecure get non-existing booking fails", testInsecureGetNonExisting, false},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			client, cfg, teardown := setupTest(t, nil, test.tls)
			defer teardown()
			test.fn(t, client, cfg)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config), secure bool) (
	client api.BookingServiceClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: config.ClientCertFile,
		KeyFile:  config.ClientKeyFile,
		CAFile:   config.CAFile,
	})
	require.NoError(t, err)

	clientCreds := insecure.NewCredentials()
	if secure {
		clientCreds = credentials.NewTLS(clientTLSConfig)
	}
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
		Server:        true,
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("no booking found for UUID: %s", u))
}

func testInsecureGetNonExisting(t *testing.T, client api.BookingServiceClient, _ *Config) {
	ctx := context.Background()
	u := uuid.New().String()

	_, err := client.GetBooking(ctx, &api.GetBookingRequest{Uuid: u})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection error: desc = \"error reading server preface: EOF\"")
}
