package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/core/ports/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDispatchMessage_routesByTopic(t *testing.T) {
	t.Parallel()
	cfg := Config{TopicUpload: "up", TopicDelete: "del"}
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockGroupHandler(ctrl)
	ctx := context.Background()
	payload := []byte(`{}`)

	mock.EXPECT().OnUpload(ctx, payload).Return(nil)
	mock.EXPECT().OnDelete(ctx, payload).Return(nil)

	require.NoError(t, dispatchMessage("up", cfg, mock, ctx, payload))
	require.NoError(t, dispatchMessage("del", cfg, mock, ctx, payload))
}

func TestDispatchMessage_unknownTopic(t *testing.T) {
	t.Parallel()
	cfg := Config{TopicUpload: "up", TopicDelete: "del"}
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockGroupHandler(ctrl)
	err := dispatchMessage("other", cfg, mock, context.Background(), []byte{})
	require.ErrorIs(t, err, ErrUnknownTopic)
}

func TestDispatchMessage_propagatesHandlerErrors(t *testing.T) {
	t.Parallel()
	cfg := Config{TopicUpload: "up", TopicDelete: "del"}
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockGroupHandler(ctrl)
	boom := errors.New("boom")
	mock.EXPECT().OnUpload(gomock.Any(), gomock.Any()).Return(boom)
	err := dispatchMessage("up", cfg, mock, context.Background(), []byte{})
	require.ErrorIs(t, err, boom)
}
