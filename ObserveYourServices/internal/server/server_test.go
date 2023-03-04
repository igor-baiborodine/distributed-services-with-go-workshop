package server

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"os"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/examples/exporter"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/api/v1"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/auth"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/config"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/internal/log"
)

var debug = flag.Bool("debug", false, "Enable observability for debugging.")

func TestMain(m *testing.M) {
	flag.Parse()
	if *debug {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	}
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	tests := []struct {
		desc string
		fn   func(t *testing.T, rootClient api.LogClient,
			nobodyClient api.LogClient, cfg *Config)
	}{
		{"produce/consume a record to/from the log succeeds", testProduceConsume},
		{"produce/consume stream succeeds", testProduceConsumeStream},
		{"consume past log boundary fails", testConsumePastBoundary},
		{"create/get a booking to/from the log succeeds", testCreateGetBooking},
		{"create/update a booking to/from the log succeeds", testCreateUpdateBooking},
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
	rootClient api.LogClient,
	nobodyClient api.LogClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	newClient := func(crtPath, keyPath string) (
		*grpc.ClientConn,
		api.LogClient,
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
		client := api.NewLogClient(conn)
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

	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	bl, err := log.NewBookingLog(dir, log.Config{})
	require.NoError(t, err)
	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)

	var telemetryExporter *exporter.LogExporter
	if *debug {
		metricsLogFile, err := os.CreateTemp("", "metrics-*.log")
		require.NoError(t, err)
		t.Logf("metrics log file: %s", metricsLogFile.Name())

		tracesLogFile, err := os.CreateTemp("", "traces-*.log")
		require.NoError(t, err)
		t.Logf("traces log file: %s", tracesLogFile.Name())

		telemetryExporter, err = exporter.NewLogExporter(exporter.Options{
			MetricsLogFile:    metricsLogFile.Name(),
			TracesLogFile:     tracesLogFile.Name(),
			ReportingInterval: time.Second,
		})
		require.NoError(t, err)
		err = telemetryExporter.Start()
		require.NoError(t, err)
	}

	cfg = &Config{
		BookingLog: bl,
		Authorizer: authorizer,
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

		if telemetryExporter != nil {
			time.Sleep(1500 * time.Millisecond)
			telemetryExporter.Stop()
			telemetryExporter.Close()
		}
	}
}

func testProduceConsume(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
	ctx := context.Background()
	b := newRandomBooking(t)
	want := newRecord(t, b)

	produce, err := rootClient.Produce(
		ctx,
		&api.ProduceRequest{
			Record: want,
		},
	)
	require.NoError(t, err)

	consume, err := rootClient.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

func testConsumePastBoundary(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
	ctx := context.Background()
	b := newRandomBooking(t)
	produce, err := rootClient.Produce(
		ctx, &api.ProduceRequest{Record: newRecord(t, b)})
	require.NoError(t, err)

	consume, err := rootClient.Consume(ctx, &api.ConsumeRequest{
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

func testProduceConsumeStream(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
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
		stream, err := rootClient.ProduceStream(ctx)
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
		stream, err := rootClient.ConsumeStream(
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

func testCreateGetBooking(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
	ctx := context.Background()
	want := newRandomBooking(t)
	created, err := rootClient.CreateBooking(
		ctx,
		&api.CreateBookingRequest{
			Booking: want,
		},
	)
	require.NoError(t, err)
	want.CreatedAt = created.Booking.CreatedAt

	got, err := rootClient.GetBooking(ctx, &api.GetBookingRequest{
		Uuid: want.Uuid,
	})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
	require.Nil(t, got.Booking.UpdatedAt)
}

func testCreateUpdateBooking(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
	ctx := context.Background()
	want := newRandomBooking(t)
	created, err := rootClient.CreateBooking(
		ctx,
		&api.CreateBookingRequest{
			Booking: want,
		},
	)
	require.NoError(t, err)
	want.CreatedAt = created.Booking.CreatedAt
	want.StartDate = "2023-02-15"
	want.EndDate = "2023-02-18"

	got, err := rootClient.UpdateBooking(ctx, &api.UpdateBookingRequest{
		Booking: want,
	})
	require.NoError(t, err)
	assertBooking(t, want, got.Booking)
	require.NotNil(t, got.Booking.UpdatedAt)
}

func testGetNonExisting(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
	ctx := context.Background()
	u := uuid.NewString()
	got, err := rootClient.GetBooking(ctx, &api.GetBookingRequest{
		Uuid: u,
	})
	require.Nilf(t, got, "get booking response should be nil")
	require.Errorf(t, err, "no booking found for UUID: %s", u)
}

func testUnauthorized(t *testing.T, rootClient api.LogClient,
	nobodyClient api.LogClient, config *Config) {
	ctx := context.Background()
	got, err := nobodyClient.GetBooking(ctx,
		&api.GetBookingRequest{Uuid: uuid.New().String()})

	require.Nilf(t, got, "get booking response should be nil")
	require.Error(t, err)

	s := status.Convert(err)
	assert.Equal(t, s.Code(), codes.PermissionDenied)
	assert.Equal(t, s.Message(), "nobody not permitted to getBooking to *")
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
