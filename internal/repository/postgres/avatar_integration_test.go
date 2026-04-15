package postgres

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/arvaliullin/goph-profile/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
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

func TestAvatarRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("docker")
	}
	if !dockerAvailable() {
		t.Skip("docker is not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pg, err := tcpostgres.Run(ctx, "postgres:18-alpine",
		tcpostgres.WithDatabase("avatars"),
		tcpostgres.WithUsername("avatars"),
		tcpostgres.WithPassword("avatars"),
	)
	if err != nil {
		t.Skipf("postgres container: %v", err)
	}
	defer func() {
		if terr := pg.Terminate(context.WithoutCancel(ctx)); terr != nil {
			t.Errorf("terminate postgres: %v", terr)
		}
	}()

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	var db *DB
	for range 60 {
		if err = RunMigrations(ctx, dsn); err != nil {
			select {
			case <-ctx.Done():
				require.NoError(t, err)
			case <-time.After(400 * time.Millisecond):
			}
			continue
		}
		db, err = New(ctx, dsn)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			require.NoError(t, err)
		case <-time.After(400 * time.Millisecond):
		}
	}
	require.NoError(t, err)
	defer db.Close()

	repo := NewAvatarRepository(db.Pool)
	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	a := &domain.Avatar{
		ID:               id,
		UserID:           "user-int",
		FileName:         "a.png",
		MimeType:         "image/png",
		SizeBytes:        10,
		S3Key:            "avatars/user-int/" + id.String() + "/original",
		ThumbnailS3Keys:  map[string]string{},
		UploadStatus:     domain.UploadStatusCompleted,
		ProcessingStatus: domain.ProcessingStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, repo.Create(ctx, a))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, a.UserID, got.UserID)
	require.Equal(t, a.S3Key, got.S3Key)

	latest, err := repo.GetLatestByUserID(ctx, "user-int")
	require.NoError(t, err)
	require.Equal(t, id, latest.ID)

	list, err := repo.ListByUserID(ctx, "user-int")
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, repo.UpdateProcessingStatus(ctx, id, domain.ProcessingStatusProcessing))
	require.NoError(t, repo.UpdateOriginalDimensions(ctx, id, 800, 600))
	require.NoError(t, repo.UpdateThumbnailKeys(ctx, id, map[string]string{domain.Thumbnail100: "t1"}))

	ok, err := repo.SoftDelete(ctx, id, "other")
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = repo.SoftDelete(ctx, id, "user-int")
	require.NoError(t, err)
	require.True(t, ok)

	_, err = repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domain.ErrNotFound)

	all, err := repo.GetByIDIncludingDeleted(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, all.DeletedAt)
}
