package minio

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

func TestStorage_PutGetDeleteMany_idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("docker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	mc, err := tminio.Run(ctx, "minio/minio:latest")
	if err != nil {
		t.Skipf("minio container: %v", err)
	}
	defer func() { _ = mc.Terminate(context.WithoutCancel(ctx)) }()

	ep, err := mc.ConnectionString(ctx)
	require.NoError(t, err)

	bucket := "it-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	st, err := New(ctx, ep, mc.Username, mc.Password, false, bucket)
	require.NoError(t, err)

	key := "obj/integration"
	payload := []byte{9, 8, 7}
	require.NoError(t, st.Put(ctx, key, bytes.NewReader(payload), int64(len(payload)), "application/octet-stream"))
	require.NoError(t, st.Ping(ctx))

	rc, err := st.Get(ctx, key)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	require.NoError(t, err)
	require.Equal(t, payload, got)

	require.NoError(t, st.DeleteMany(ctx, []string{key, key}))
	require.NoError(t, st.DeleteMany(ctx, []string{key}))

	rc2, err := st.Get(ctx, key)
	if err == nil {
		_, err = io.ReadAll(rc2)
		_ = rc2.Close()
	}
	require.Error(t, err)
}
