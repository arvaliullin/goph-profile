package worker

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/core/ports/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleDelete(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	st.EXPECT().DeleteMany(gomock.Any(), []string{"a", "b"}).Return(nil)
	p := NewProcessor(repo, st, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ev := ports.AvatarDeleteEvent{AvatarID: "x", S3Keys: []string{"a", "b"}}
	b, err := json.Marshal(ev)
	require.NoError(t, err)
	require.NoError(t, p.HandleDelete(context.Background(), b))
}
