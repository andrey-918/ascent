package store_test

import (
	"ascent/fixtures"
	"ascent/store"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDiaryEntryStore(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})
	now := time.Now()
	ctx := context.Background()
	userStore := store.NewUserStore(env.Db)
	user, err := userStore.CreateUser(context.Background(), "testingDiaryEntries@test.com", "testingpassword")
	require.NoError(t, err)

	diaryStore := store.NewDiaryEntryStore(env.Db)
	diaryEntry, err := diaryStore.Create(ctx, user.Id, "testing content")
	require.NoError(t, err)

	require.Equal(t, diaryEntry.UserID, user.Id)
	require.Equal(t, diaryEntry.Content, "testing content")
	require.Less(t, now.UnixNano(), diaryEntry.CreatedAt.UnixNano())

	diaryEntries, err := diaryStore.ListByUser(ctx, user.Id, 10, 0)
	require.NoError(t, err)
	require.Equal(t, len(diaryEntries), 1)

	diaryEntry2, err := diaryStore.Create(ctx, user.Id, "testing content N2")
	require.NoError(t, err)

	diaryEntries, err = diaryStore.ListByUser(ctx, user.Id, 10, 0)
	require.NoError(t, err)
	require.Equal(t, len(diaryEntries), 2)

	diaryEntry, err = diaryStore.ById(ctx, diaryEntry2.ID, user.Id)
	require.NoError(t, err)
	require.Equal(t, diaryEntry.ID, diaryEntry2.ID)
	require.Equal(t, diaryEntry.UserID, diaryEntry2.UserID)
	require.Equal(t, diaryEntry.Content, diaryEntry2.Content)
	require.Equal(t, diaryEntry.CreatedAt, diaryEntry2.CreatedAt)

	err = diaryStore.Delete(ctx, diaryEntry2.ID, user.Id)
	require.NoError(t, err)

	diaryEntry, err = diaryStore.ById(ctx, diaryEntry2.ID, user.Id)
	require.ErrorIs(t, err, store.ErrDiaryEntryNotFound)
}

func TestDiaryEntryStorePagination(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	ctx := context.Background()
	userStore := store.NewUserStore(env.Db)
	user, err := userStore.CreateUser(ctx, "testingDiaryPagination@test.com", "testingpassword")
	require.NoError(t, err)

	diaryStore := store.NewDiaryEntryStore(env.Db)
	for _, content := range []string{"entry 1", "entry 2", "entry 3"} {
		_, err := diaryStore.Create(ctx, user.Id, content)
		require.NoError(t, err)
	}

	allEntries, err := diaryStore.ListByUser(ctx, user.Id, 10, 0)
	require.NoError(t, err)
	require.Len(t, allEntries, 3)

	firstPage, err := diaryStore.ListByUser(ctx, user.Id, 2, 0)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	require.Equal(t, allEntries[:2], firstPage)

	secondPage, err := diaryStore.ListByUser(ctx, user.Id, 2, 2)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	require.Equal(t, allEntries[2:], secondPage)
}
