package server

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/store"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api.BookingServiceClient,
		config *Config,
	){
		"create/get booking succeeds":    testCreateGet,
		"get non-existing booking fails": testGetNonExisting,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (
	client api.BookingServiceClient,
	config *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	clientOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	cc, err := grpc.Dial(l.Addr().String(), clientOptions...)
	require.NoError(t, err)

	bs, err := store.NewBookingStore()
	require.NoError(t, err)

	config = &Config{
		BookingStore: bs,
	}
	if fn != nil {
		fn(config)
	}
	server, err := NewGRPCServer(config)
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	client = api.NewBookingServiceClient(cc)

	return client, config, func() {
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
