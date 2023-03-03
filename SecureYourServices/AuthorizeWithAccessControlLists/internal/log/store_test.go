package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/api/v1"
)

func TestStoreAppendRead(t *testing.T) {
	f, err := os.CreateTemp("", "store_append_read_test")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	s, err := newStore(f)
	require.NoError(t, err)
	b := newRandomBooking(t)
	r := newRecord(t, b)

	testAppend(t, s, r)
	testRead(t, s, r)
	testReadAt(t, s, r)

	s, err = newStore(f)
	require.NoError(t, err)
	testRead(t, s, r)
}

func testAppend(t *testing.T, s *store, r *api.Record) {
	t.Helper()
	for i := uint64(1); i < 4; i++ {
		n, pos, err := s.Append(r.Value)
		require.NoError(t, err)
		width := uint64(len(r.Value)) + lenWidth
		require.Equal(t, pos+n, width*i)
	}
}

func testRead(t *testing.T, s *store, r *api.Record) {
	t.Helper()
	var pos uint64

	for i := uint64(1); i < 4; i++ {
		read, err := s.Read(pos)
		require.NoError(t, err)
		require.Equal(t, r.Value, read)
		width := uint64(len(r.Value)) + lenWidth
		pos += width
	}
}

func testReadAt(t *testing.T, s *store, r *api.Record) {
	t.Helper()
	for i, off := uint64(1), int64(0); i < 4; i++ {
		b := make([]byte, lenWidth)
		n, err := s.ReadAt(b, off)
		require.NoError(t, err)
		require.Equal(t, lenWidth, n)
		off += int64(n)

		size := enc.Uint64(b)
		b = make([]byte, size)
		n, err = s.ReadAt(b, off)
		require.NoError(t, err)
		require.Equal(t, r.Value, b)
		require.Equal(t, int(size), n)
		off += int64(n)
	}
}

func TestStoreClose(t *testing.T) {
	f, err := os.CreateTemp("", "store_close_test")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	s, err := newStore(f)
	require.NoError(t, err)

	b := newRandomBooking(t)
	r := newRecord(t, b)
	_, _, err = s.Append(r.Value)
	require.NoError(t, err)

	f, beforeSize, err := openFile(f.Name())
	require.NoError(t, err)

	err = s.Close()
	require.NoError(t, err)

	f, afterSize, err := openFile(f.Name())
	require.True(t, afterSize > beforeSize)
}

func openFile(name string) (file *os.File, size int64, err error) {
	f, err := os.OpenFile(
		name,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, 0, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	return f, fi.Size(), nil
}
