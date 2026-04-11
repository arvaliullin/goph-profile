package kafka

import (
	"testing"

	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadEventRoundTrip(t *testing.T) {
	want := ports.AvatarUploadEvent{AvatarID: "id-1", UserID: "u", S3Key: "avatars/u/id/original"}
	b, err := MarshalUploadEvent(want)
	require.NoError(t, err)
	got, err := UnmarshalUploadEvent(b)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDeleteEventRoundTrip(t *testing.T) {
	want := ports.AvatarDeleteEvent{AvatarID: "id-1", S3Keys: []string{"a", "b"}}
	b, err := MarshalDeleteEvent(want)
	require.NoError(t, err)
	got, err := UnmarshalDeleteEvent(b)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
