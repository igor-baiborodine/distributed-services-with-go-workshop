package log

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ObserveYourSystems/api/v1"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T, log *Log,
	){
		"append and read a record succeeds":  testAppendRead,
		"offset out of range error":          testOutOfRangeErr,
		"init with existing segments":        testInitExisting,
		"reader":                             testReader,
		"truncate":                           testTruncate,
		"append and read a booking succeeds": testAppendReadBooking,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "store-test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			c := Config{}
			c.Segment.MaxStoreBytes = 1024
			log, err := NewBookingLog(dir, c)
			require.NoError(t, err)

			fn(t, log)
		})
	}
}

func testAppendRead(t *testing.T, log *Log) {
	b := newRandomBooking(t)
	want := newRecord(t, b)
	off, err := log.Append(want)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	got, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, want.Value, got.Value)
	require.Equal(t, uint64(0), got.Offset)
}

func testAppendReadBooking(t *testing.T, log *Log) {
	want := newRandomBooking(t)
	off, err := log.AppendBooking(want)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	got, err := log.ReadBooking(want.Uuid)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func testOutOfRangeErr(t *testing.T, log *Log) {
	got, err := log.Read(1)
	require.Nil(t, got)
	require.Error(t, err)
}

func testInitExisting(t *testing.T, log *Log) {
	for i := 0; i < 3; i++ {
		b := newRandomBooking(t)
		r := newRecord(t, b)
		_, err := log.Append(r)
		require.NoError(t, err)
	}
	require.NoError(t, log.Close())

	got, err := log.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), got)
	got, err = log.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), got)

	n, err := NewBookingLog(log.Dir, log.Config)
	require.NoError(t, err)

	got, err = n.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), got)
	got, err = n.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), got)
	require.Equal(t, 3, len(log.activeSegment.uuids))
}

func testReader(t *testing.T, log *Log) {
	b := newRandomBooking(t)
	want := newRecord(t, b)
	off, err := log.Append(want)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := log.Reader()
	out, err := io.ReadAll(reader)
	require.NoError(t, err)

	got := &api.Record{}
	err = proto.Unmarshal(out[lenWidth:], got)
	require.NoError(t, err)
	require.Equal(t, want.Value, got.Value)
}

func testTruncate(t *testing.T, log *Log) {
	for i := 0; i < 2; i++ {
		b := newRandomBooking(t)
		r := newRecord(t, b)
		_, err := log.Append(r)
		require.NoError(t, err)
	}
	err := log.Truncate(1)
	require.NoError(t, err)

	_, err = log.Read(0)
	require.Error(t, err)
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
