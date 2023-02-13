package server

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/internal/config"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/internal/store"
)

func TestServer(t *testing.T) {
	tests := []struct {
		desc string
		fn   func(t *testing.T, client api.BookingServiceClient, cfg *Config)
		tls  bool
	}{
		{"create and get by UUID booking succeeds", testCreateGetByUUID, true},
		{"create and get by ID booking succeeds", testCreateGetByID, true},
		{"create and update booking succeeds", testCreateUpdate, true},
		{"create and get booking stream succeeds",
			testCreateGetBookingStream, true},
		{"get non-existing booking fails",
			testGetNonExisting, true},
		{"insecure fails", testInsecureGetNonExisting, false},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			client, cfg, teardown := setupTest(t, nil, test.tls)
			defer teardown()
			test.fn(t, client, cfg)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config), tls bool) (
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

	clientCreds := insecure.NewCredentials()
	if tls {
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

func testCreateGetByUUID(t *testing.T, client api.BookingServiceClient,
	config *Config) {
	ctx := context.Background()
	want, err := createBooking(client, ctx)
	require.NoError(t, err)

	got, err := client.GetBookingByUUID(ctx,
		&api.GetByUUIDBookingRequest{UUID: want.UUID})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
}

func testCreateGetByID(t *testing.T, client api.BookingServiceClient,
	config *Config) {
	ctx := context.Background()
	want, err := createBooking(client, ctx)
	require.NoError(t, err)

	got, err := client.GetBookingByID(ctx, &api.GetByIDBookingRequest{ID: 1})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
}

func testCreateUpdate(t *testing.T, client api.BookingServiceClient,
	config *Config) {
	ctx := context.Background()
	want, err := createBooking(client, ctx)
	require.NoError(t, err)
	want.StartDate = "2023-02-15"
	want.EndDate = "2023-02-18"

	got, err := client.UpdateBooking(ctx,
		&api.UpdateBookingRequest{Booking: want})
	require.NoError(t, err)
	want.ID++
	assertBooking(t, want, got.Booking)
}

func testGetNonExisting(t *testing.T, client api.BookingServiceClient,
	config *Config) {
	ctx := context.Background()
	u := uuid.New().String()
	_, err := client.GetBookingByUUID(ctx, &api.GetByUUIDBookingRequest{UUID: u})
	require.Errorf(t, err, "no booking found for UUID: %s", u)
}

func testCreateGetBookingStream(t *testing.T, client api.BookingServiceClient,
	config *Config) {
	booking1 := createBookingProto(1, "john.smith@dot.com", "John Smith",
		"2023-01-20", "2023-01-23")
	booking2 := createBookingProto(2, "jack.jones@dot.com", "Jack Jones",
		"2023-01-27", "2023-01-30")
	booking3 := createBookingProto(3, "robert.brown@dot.com", "Robert Brown",
		"2023-02-11", "2023-02-15")

	var bookings = []*api.Booking{booking1, booking2, booking3}
	ctx := context.Background()

	{
		stream, err := client.CreateBookingStream(ctx)
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
		stream, err := client.GetBookingStream(
			ctx, &api.GetByIDBookingRequest{ID: 1})
		require.NoError(t, err)

		for _, b := range bookings {
			res, err := stream.Recv()
			require.NoError(t, err)
			assertBooking(t, b, res.Booking)
		}
	}
}

func testInsecureGetNonExisting(t *testing.T, client api.BookingServiceClient,
	config *Config) {
	ctx := context.Background()
	got, err := client.GetBookingByUUID(ctx,
		&api.GetByUUIDBookingRequest{UUID: uuid.New().String()})
	require.Nilf(t, got, "get booking response should be nil")
	require.Error(t, err)

	gotCode, wantCode := status.Code(err), codes.Unavailable
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
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
