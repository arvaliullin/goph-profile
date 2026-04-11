package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubHandler struct {
	uploadCalls int
	deleteCalls int
	uploadErr   error
	deleteErr   error
}

func (s *stubHandler) OnUpload(_ context.Context, _ []byte) error {
	s.uploadCalls++
	return s.uploadErr
}

func (s *stubHandler) OnDelete(_ context.Context, _ []byte) error {
	s.deleteCalls++
	return s.deleteErr
}

func TestDispatchMessage_routesByTopic(t *testing.T) {
	t.Parallel()
	cfg := Config{TopicUpload: "up", TopicDelete: "del"}
	h := &stubHandler{}
	ctx := context.Background()
	payload := []byte(`{}`)

	require.NoError(t, dispatchMessage("up", cfg, h, ctx, payload))
	require.Equal(t, 1, h.uploadCalls)
	require.Equal(t, 0, h.deleteCalls)

	require.NoError(t, dispatchMessage("del", cfg, h, ctx, payload))
	require.Equal(t, 1, h.uploadCalls)
	require.Equal(t, 1, h.deleteCalls)
}

func TestDispatchMessage_unknownTopic(t *testing.T) {
	t.Parallel()
	cfg := Config{TopicUpload: "up", TopicDelete: "del"}
	h := &stubHandler{}
	err := dispatchMessage("other", cfg, h, context.Background(), []byte{})
	require.ErrorIs(t, err, ErrUnknownTopic)
}

func TestDispatchMessage_propagatesHandlerErrors(t *testing.T) {
	t.Parallel()
	cfg := Config{TopicUpload: "up", TopicDelete: "del"}
	h := &stubHandler{uploadErr: errors.New("boom")}
	err := dispatchMessage("up", cfg, h, context.Background(), []byte{})
	require.ErrorIs(t, err, h.uploadErr)
}
