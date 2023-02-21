package log

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOffIndex(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "offIndex_test")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	c := Config{}
	c.Segment.MaxIndexBytes = 1024
	idx, err := newIndex(f, offIndexWidth, c)
	require.NoError(t, err)
	offIndex := &offIndex{idx}
	_, _, err = offIndex.Read(-1)
	require.Error(t, err)
	require.Equal(t, f.Name(), offIndex.Name())

	entries := []struct {
		Off uint32
		Pos uint64
	}{
		{Off: 0, Pos: 0},
		{Off: 1, Pos: 10},
	}

	for _, want := range entries {
		err = offIndex.Write(want.Off, want.Pos)
		require.NoError(t, err)

		_, pos, err := offIndex.Read(int64(want.Off))
		require.NoError(t, err)
		require.Equal(t, want.Pos, pos)
	}

	// offIndex and scanner should error when reading past existing entries
	_, _, err = offIndex.Read(int64(len(entries)))
	require.Equal(t, io.EOF, err)
	_ = offIndex.Close()

	// offIndex should build its state from the existing file
	f, _ = os.OpenFile(f.Name(), os.O_RDWR, 0600)
	idx, err = newIndex(f, offIndexWidth, c)
	require.NoError(t, err)
	off, pos, err := offIndex.Read(-1)
	require.NoError(t, err)
	require.Equal(t, uint32(1), off)
	require.Equal(t, entries[1].Pos, pos)
}
