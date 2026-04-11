package minio

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_InvalidEndpoint(t *testing.T) {
	t.Parallel()
	_, err := New(context.Background(), "127.0.0.1:1", "k", "s", false, "b")
	require.Error(t, err)
}

func TestPutGetDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, err := New(ctx, "play.min.io", "", "", true, "avatars-test-bucket-"+t.Name())
	if err != nil {
		t.Skip("no network or minio demo unavailable")
	}
	key := "unit/" + t.Name()
	payload := []byte{1, 2, 3}
	require.NoError(t, st.Put(ctx, key, bytes.NewReader(payload), int64(len(payload)), "application/octet-stream"))
	rc, err := st.Get(ctx, key)
	require.NoError(t, err)
	got, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	require.NoError(t, rerr)
	require.NoError(t, cerr)
	require.Equal(t, payload, got)
	require.NoError(t, st.Delete(ctx, key))
	require.NoError(t, st.DeleteMany(ctx, []string{key}))
	require.NoError(t, st.Ping(ctx))
}
