package avatar

import (
	"bytes"
	"context"
	"image/color"
	"io"
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/arvaliullin/goph-profile/internal/core/ports"
	"github.com/arvaliullin/goph-profile/internal/core/ports/mocks"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// attachMemObjectStorage подключает in-memory хранилище к моку ObjectStorage (Put/Get).
func attachMemObjectStorage(st *mocks.MockObjectStorage, data map[string][]byte) {
	wild := gomock.Any()
	st.EXPECT().Put(wild, wild, wild, wild, wild).DoAndReturn(
		func(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
			b, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			data[key] = b
			return nil
		}).AnyTimes()
	st.EXPECT().Get(wild, wild).DoAndReturn(
		func(_ context.Context, key string) (io.ReadCloser, error) {
			b, ok := data[key]
			if !ok {
				return nil, domain.ErrNotFound
			}
			return io.NopCloser(bytes.NewReader(b)), nil
		}).AnyTimes()
	st.EXPECT().Delete(wild, wild).AnyTimes()
	st.EXPECT().DeleteMany(wild, wild).AnyTimes()
}

func TestUploadMissingUser(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)
	svc := New(repo, st, pub, clk, 10<<20)
	_, err := svc.Upload(context.Background(), "", "a.jpg", "image/jpeg", bytes.NewReader([]byte{1}), 1)
	require.ErrorIs(t, err, domain.ErrMissingUserID)
}

func TestUploadTooLarge(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)
	svc := New(repo, st, pub, clk, 5)
	_, err := svc.Upload(context.Background(), "u", "a.jpg", "image/jpeg", bytes.NewReader(bytes.Repeat([]byte{'a'}, 10)), 10)
	require.ErrorIs(t, err, domain.ErrFileTooLarge)
}

func TestUploadHappyPath(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	data := map[string][]byte{}
	attachMemObjectStorage(st, data)

	img := imaging.New(16, 16, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.JPEG))

	fixed := time.Unix(1, 0).UTC()
	clk.EXPECT().Now().Return(fixed)

	repo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&domain.Avatar{})).Return(nil)

	var gotEvent ports.AvatarUploadEvent
	pub.EXPECT().PublishUpload(gomock.Any(), gomock.Any()).Do(
		func(_ context.Context, e ports.AvatarUploadEvent) { gotEvent = e },
	).Return(nil)

	svc := New(repo, st, pub, clk, 10<<20)

	a, err := svc.Upload(context.Background(), "u1", "a.jpg", "image/jpeg", bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Equal(t, "u1", a.UserID)
	require.Equal(t, a.ID.String(), gotEvent.AvatarID)
	require.Equal(t, "u1", gotEvent.UserID)
	require.Contains(t, gotEvent.S3Key, "u1")
	require.Contains(t, gotEvent.S3Key, a.ID.String())
}

func TestMetadataAndList(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	av := &domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: 10, S3Key: "k",
		ThumbnailS3Keys: map[string]string{
			domain.Thumbnail100: "t100",
		},
		UploadStatus: domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Unix(2, 0).UTC(), UpdatedAt: time.Unix(3, 0).UTC(),
	}
	repo.EXPECT().GetByID(gomock.Any(), id).Return(av, nil)
	repo.EXPECT().ListByUserID(gomock.Any(), "u").Return([]domain.Avatar{*av}, nil)

	data := map[string][]byte{"k": {1, 2, 3}}
	attachMemObjectStorage(st, data)

	svc := New(repo, st, pub, clk, 10<<20)

	m, err := svc.Metadata(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, id.String(), m.ID.String())

	list, err := svc.ListMetadata(context.Background(), "u")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestGetImageOriginalAndFormat(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	img := imaging.New(8, 8, color.RGBA{A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.PNG))
	id := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.png", MimeType: "image/png",
		SizeBytes: int64(buf.Len()), S3Key: "k",
		ThumbnailS3Keys: map[string]string{},
		UploadStatus:    domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	data := map[string][]byte{"k": append([]byte(nil), buf.Bytes()...)}
	attachMemObjectStorage(st, data)

	svc := New(repo, st, pub, clk, 10<<20)

	rc, mime, etag, err := svc.GetImage(context.Background(), id, domain.SizeOriginal, "jpeg")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)
	require.NotEmpty(t, etag)
	b, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	require.NoError(t, rerr)
	require.NoError(t, cerr)
	require.NotEmpty(t, b)
}

func TestGetImageForUser(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	img := imaging.New(4, 4, color.RGBA{R: 1, A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.JPEG))
	id := uuid.New()
	repo.EXPECT().GetLatestByUserID(gomock.Any(), "u").Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: int64(buf.Len()), S3Key: "k",
		UploadStatus: domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	data := map[string][]byte{"k": buf.Bytes()}
	attachMemObjectStorage(st, data)

	svc := New(repo, st, pub, clk, 10<<20)

	rc, mime, _, err := svc.GetImageForUser(context.Background(), "u")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)
	_, rerr := io.ReadAll(rc)
	require.NoError(t, rerr)
	require.NoError(t, rc.Close())
}

func TestDeleteOK(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: 1, S3Key: "k", ThumbnailS3Keys: map[string]string{domain.Thumbnail100: "t"},
		UploadStatus: domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)
	repo.EXPECT().SoftDelete(gomock.Any(), id, "u").Return(true, nil)

	var del ports.AvatarDeleteEvent
	pub.EXPECT().PublishDelete(gomock.Any(), gomock.Any()).Do(
		func(_ context.Context, e ports.AvatarDeleteEvent) { del = e },
	).Return(nil)

	svc := New(repo, st, pub, clk, 10<<20)
	require.NoError(t, svc.Delete(context.Background(), id, "u"))
	require.Equal(t, id.String(), del.AvatarID)
	require.Contains(t, del.S3Keys, "k")
	require.Contains(t, del.S3Keys, "t")
}

func TestDeleteForUser(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	repo.EXPECT().GetLatestByUserID(gomock.Any(), "u").Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: 1, S3Key: "k",
		UploadStatus: domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: 1, S3Key: "k",
		UploadStatus: domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)
	repo.EXPECT().SoftDelete(gomock.Any(), id, "u").Return(true, nil)
	pub.EXPECT().PublishDelete(gomock.Any(), gomock.Any()).Return(nil)

	svc := New(repo, st, pub, clk, 10<<20)
	require.NoError(t, svc.DeleteForUser(context.Background(), "u", "u"))
}

func TestGetImageThumbFormatTranscode(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	img := imaging.New(16, 16, color.RGBA{A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.JPEG))
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: int64(buf.Len()), S3Key: "k",
		ThumbnailS3Keys: map[string]string{domain.Thumbnail300: "tk"},
		UploadStatus:    domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	data := map[string][]byte{"tk": buf.Bytes()}
	attachMemObjectStorage(st, data)

	svc := New(repo, st, pub, clk, 10<<20)

	rc, mime, _, err := svc.GetImage(context.Background(), id, domain.Thumbnail300, "png")
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)
	_, rerr := io.ReadAll(rc)
	require.NoError(t, rerr)
	require.NoError(t, rc.Close())
}

func TestGetImageThumbOK(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	img := imaging.New(16, 16, color.RGBA{A: 255})
	var buf bytes.Buffer
	require.NoError(t, imaging.Encode(&buf, img, imaging.JPEG))
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: int64(buf.Len()), S3Key: "k",
		ThumbnailS3Keys: map[string]string{domain.Thumbnail100: "tk"},
		UploadStatus:    domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	data := map[string][]byte{"k": buf.Bytes(), "tk": buf.Bytes()}
	attachMemObjectStorage(st, data)

	svc := New(repo, st, pub, clk, 10<<20)

	rc, mime, _, err := svc.GetImage(context.Background(), id, domain.Thumbnail100, "")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)
	_, rerr := io.ReadAll(rc)
	require.NoError(t, rerr)
	require.NoError(t, rc.Close())
}

func TestGetImageThumbMissing404(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "u", FileName: "a.jpg", MimeType: "image/jpeg",
		SizeBytes: 3, S3Key: "k", ThumbnailS3Keys: map[string]string{},
		UploadStatus: domain.UploadStatusCompleted, ProcessingStatus: domain.ProcessingStatusCompleted,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	svc := New(repo, st, pub, clk, 10<<20)
	_, _, _, err := svc.GetImage(context.Background(), id, domain.Thumbnail100, "")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteForbidden(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	st := mocks.NewMockObjectStorage(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	clk := mocks.NewMockClock(ctrl)

	id := uuid.New()
	repo.EXPECT().GetByID(gomock.Any(), id).Return(&domain.Avatar{
		ID: id, UserID: "a", FileName: "x", MimeType: "image/jpeg",
		SizeBytes: 1, S3Key: "k", UploadStatus: domain.UploadStatusCompleted,
		ProcessingStatus: domain.ProcessingStatusPending,
		CreatedAt:        time.Now(), UpdatedAt: time.Now(),
	}, nil)

	svc := New(repo, st, pub, clk, 10<<20)
	err := svc.Delete(context.Background(), id, "other")
	require.ErrorIs(t, err, domain.ErrForbidden)
}
