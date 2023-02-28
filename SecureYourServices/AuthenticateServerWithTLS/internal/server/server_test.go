package server

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/config"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthenticateServerWithTLS/internal/log"
)

func TestServer(t *testing.T) {
	tests := []struct {
		desc string
		fn   func(t *testing.T, client api.LogClient, cfg *Config)
		tls  bool
	}{
		{"produce/consume a record to/from the log succeeds",
			testProduceConsume, true},
		{"produce/consume stream succeeds",
			testProduceConsumeStream, true},
		{"consume past log boundary fails",
			testConsumePastBoundary, true},
		{"create/get a booking to/from the log succeeds",
			testCreateGetBooking, true},
		{"create/update a booking to/from the log succeeds",
			testCreateUpdateBooking, true},
		{"get non-existing booking fails", testGetNonExisting, true},
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
	client api.LogClient,
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

	client = api.NewLogClient(cc)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	bl, err := log.NewBookingLog(dir, log.Config{})
	require.NoError(t, err)

	cfg = &Config{
		BookingLog: bl,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
	}
}
func testProduceConsume(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()
	b := newRandomBooking(t)
	want := newRecord(t, b)

	produce, err := client.Produce(
		ctx,
		&api.ProduceRequest{
			Record: want,
		},
	)
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

func testConsumePastBoundary(
	t *testing.T,
	client api.LogClient,
	config *Config,
) {
	ctx := context.Background()
	b := newRandomBooking(t)
	produce, err := client.Produce(
		ctx, &api.ProduceRequest{Record: newRecord(t, b)})
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset + 1,
	})
	if consume != nil {
		t.Fatal("consume not nil")
	}
	got := status.Code(err)
	want := status.Code(api.NewErrNotFoundForOffset(produce.Offset + 1))
	if got != want {
		t.Fatalf("got err: %v, want: %v", got, want)
	}
}

func testProduceConsumeStream(
	t *testing.T,
	client api.LogClient,
	config *Config,
) {
	ctx := context.Background()
	b1, b2 := newRandomBooking(t), newRandomBooking(t)
	records := []*api.Record{{
		Offset: 0,
		Value:  newRecord(t, b1).Value,
	}, {
		Offset: 1,
		Value:  newRecord(t, b2).Value,
	}}

	{
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)

		for offset, record := range records {
			err = stream.Send(&api.ProduceRequest{
				Record: record,
			})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)
			if res.Offset != uint64(offset) {
				t.Fatalf(
					"got offset: %d, want: %d",
					res.Offset,
					offset,
				)
			}
		}
	}

	{
		stream, err := client.ConsumeStream(
			ctx,
			&api.ConsumeRequest{Offset: 0},
		)
		require.NoError(t, err)

		for i, record := range records {
			res, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, res.Record, &api.Record{
				Value:  record.Value,
				Offset: uint64(i),
			})
		}
	}
}

func testCreateGetBooking(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()
	want := newRandomBooking(t)
	created, err := client.CreateBooking(
		ctx,
		&api.CreateBookingRequest{
			Booking: want,
		},
	)
	require.NoError(t, err)
	want.CreatedAt = created.Booking.CreatedAt

	got, err := client.GetBooking(ctx, &api.GetBookingRequest{
		Uuid: want.Uuid,
	})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
	require.Nil(t, got.Booking.UpdatedAt)
}

func testCreateUpdateBooking(t *testing.T, client api.LogClient,
	config *Config) {
	ctx := context.Background()
	want := newRandomBooking(t)
	created, err := client.CreateBooking(
		ctx,
		&api.CreateBookingRequest{
			Booking: want,
		},
	)
	require.NoError(t, err)
	want.CreatedAt = created.Booking.CreatedAt
	want.StartDate = "2023-02-15"
	want.EndDate = "2023-02-18"

	got, err := client.UpdateBooking(ctx, &api.UpdateBookingRequest{
		Booking: want,
	})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
	require.NotNil(t, got.Booking.UpdatedAt)
}

func testGetNonExisting(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()
	u := uuid.NewString()
	got, err := client.GetBooking(ctx, &api.GetBookingRequest{
		Uuid: u,
	})
	require.Nilf(t, got, "get booking response should be nil")
	require.Errorf(t, err, "no booking found for UUID: %s", u)
}

func testInsecureGetNonExisting(t *testing.T, client api.LogClient,
	config *Config) {
	ctx := context.Background()
	got, err := client.GetBooking(ctx, &api.GetBookingRequest{
		Uuid: uuid.NewString(),
	})
	require.Nilf(t, got, "get booking response should be nil")
	require.Error(t, err)

	gotCode, wantCode := status.Code(err), codes.Unavailable
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
}

func assertBooking(t *testing.T, want *api.Booking, got *api.Booking) {
	require.Equal(t, want.Uuid, got.Uuid)
	require.Equal(t, want.Email, got.Email)
	require.Equal(t, want.FullName, got.FullName)
	require.Equal(t, want.StartDate, got.StartDate)
	require.Equal(t, want.EndDate, got.EndDate)
	require.Equal(t, true, got.Active)
	require.Equal(t, want.CreatedAt.AsTime(), got.CreatedAt.AsTime())
}

func newRandomBooking(t *testing.T) *api.Booking {
	b := &api.Booking{}
	err := faker.FakeData(b)
	require.NoError(t, err)
	b.Uuid = uuid.NewString()
	return b
}

func newRecord(t *testing.T, b *api.Booking) *api.Record {
	v, err := json.Marshal(b)
	require.NoError(t, err)
	return &api.Record{Value: v}
}
