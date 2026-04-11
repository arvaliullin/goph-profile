package minio

import (
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

func TestIsNoSuchKeyOrBucket(t *testing.T) {
	t.Parallel()
	require.False(t, isNoSuchKeyOrBucket(nil))
	require.True(t, isNoSuchKeyOrBucket(minio.ErrorResponse{Code: "NoSuchKey"}))
	require.True(t, isNoSuchKeyOrBucket(minio.ErrorResponse{Code: "NoSuchBucket"}))
	require.False(t, isNoSuchKeyOrBucket(minio.ErrorResponse{Code: "SlowDown"}))
}
