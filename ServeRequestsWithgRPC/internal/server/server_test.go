package server

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/store"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client api.BookingServiceClient,
		config *Config,
	){
		"create and get by UUID booking succeeds": testCreateGetByUUID,
		"create and get by ID booking succeeds":   testCreateGetByID,
		"create and update booking succeeds":      testCreateUpdate,
		"create and get booking stream succeeds":  testCreateGetBookingStream,
		"get non-existing booking fails":          testGetNonExisting,
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

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	cc, err := grpc.Dial(l.Addr().String(), opts...)
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
