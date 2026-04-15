package minio

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

func dockerAvailable() bool {
	for _, sock := range []string{"/var/run/docker.sock", os.ExpandEnv("$HOME/.docker/run/docker.sock")} {
		conn, err := net.DialTimeout("unix", sock, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

func TestStorage_PutGetDeleteMany_idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("docker")
	}
	if !dockerAvailable() {
		t.Skip("docker is not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	mc, err := tminio.Run(ctx, "minio/minio:latest")
	if err != nil {
		t.Skipf("minio container: %v", err)
	}
	defer func() {
		if terr := mc.Terminate(context.WithoutCancel(ctx)); terr != nil {
			t.Errorf("terminate minio: %v", terr)
		}
	}()

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
	got, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	require.NoError(t, rerr)
	require.NoError(t, cerr)
	require.Equal(t, payload, got)

	require.NoError(t, st.DeleteMany(ctx, []string{key, key}))
	require.NoError(t, st.DeleteMany(ctx, []string{key}))

	rc2, err := st.Get(ctx, key)
	if err == nil {
		var rerr error
		_, rerr = io.ReadAll(rc2)
		cerr := rc2.Close()
		switch {
		case rerr != nil:
			err = rerr
		case cerr != nil:
			err = cerr
		}
	}
	require.Error(t, err)
}
