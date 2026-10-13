package store_test

import (
	"ascent/fixtures"
	"ascent/store"
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFinancialEventStoreApply(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	ctx := context.Background()
	dataStore := store.New(env.Db)
	user, err := dataStore.Users.CreateUser(ctx, "financial-events@test.com", "testingpassword")
	require.NoError(t, err)
	goal, err := dataStore.Goals.Create(ctx, user.Id, "Savings", 1_000)
	require.NoError(t, err)
	entry, err := dataStore.DiaryEntries.Create(ctx, user.Id, "Saved money")
	require.NoError(t, err)

	deposit := store.FinancialEvent{
		ID:            uuid.New(),
		UserID:        user.Id,
		GoalID:        goal.ID,
		DiaryEntryID:  entry.ID,
		Type:          store.FinancialEventDeposit,
		AmountKopecks: 500,
		Currency:      "RUB",
		SourceText:    "отложил 5 рублей",
	}
	applied, err := dataStore.FinancialEvents.Apply(ctx, deposit)
	require.NoError(t, err)
	require.True(t, applied)

	// A delivery retry is a no-op and a changed payload with the same ID is rejected.
	applied, err = dataStore.FinancialEvents.Apply(ctx, deposit)
	require.NoError(t, err)
	require.False(t, applied)
	changedDuplicate := deposit
	changedDuplicate.AmountKopecks++
	_, err = dataStore.FinancialEvents.Apply(ctx, changedDuplicate)
	require.ErrorIs(t, err, store.ErrFinancialEventConflict)

	tooLargeWithdrawal := store.FinancialEvent{
		ID:            uuid.New(),
		UserID:        user.Id,
		GoalID:        goal.ID,
		DiaryEntryID:  entry.ID,
		Type:          store.FinancialEventWithdrawal,
		AmountKopecks: 501,
		Currency:      "RUB",
		SourceText:    "снял 5 рублей 1 копейку",
	}
	_, err = dataStore.FinancialEvents.Apply(ctx, tooLargeWithdrawal)
	require.ErrorIs(t, err, store.ErrInsufficientGoalProgress)

	goal, err = dataStore.Goals.ByID(ctx, goal.ID, user.Id)
	require.NoError(t, err)
	require.EqualValues(t, 500, goal.CurrentAmountKopecks)
	require.Equal(t, "active", goal.Status)

	withdrawal := tooLargeWithdrawal
	withdrawal.ID = uuid.New()
	withdrawal.AmountKopecks = 200
	withdrawal.SourceText = "снял 2 рубля"
	applied, err = dataStore.FinancialEvents.Apply(ctx, withdrawal)
	require.NoError(t, err)
	require.True(t, applied)

	goal, err = dataStore.Goals.ByID(ctx, goal.ID, user.Id)
	require.NoError(t, err)
	require.EqualValues(t, 300, goal.CurrentAmountKopecks)

	_, err = dataStore.DiaryEntries.ById(ctx, entry.ID, user.Id)
	require.NoError(t, err)
	err = dataStore.DiaryEntries.Delete(ctx, entry.ID, user.Id)
	require.ErrorIs(t, err, store.ErrDiaryEntryHasFinancialEvents)
}

func TestFinancialEventStoreConcurrentDeposits(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	ctx := context.Background()
	dataStore := store.New(env.Db)
	user, err := dataStore.Users.CreateUser(ctx, "concurrent-events@test.com", "testingpassword")
	require.NoError(t, err)
	goal, err := dataStore.Goals.Create(ctx, user.Id, "Concurrent savings", 1_000)
	require.NoError(t, err)
	entry, err := dataStore.DiaryEntries.Create(ctx, user.Id, "Made several deposits")
	require.NoError(t, err)

	const eventCount = 10
	var wg sync.WaitGroup
	errs := make(chan error, eventCount)
	for range eventCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := dataStore.FinancialEvents.Apply(ctx, store.FinancialEvent{
				ID:            uuid.New(),
				UserID:        user.Id,
				GoalID:        goal.ID,
				DiaryEntryID:  entry.ID,
				Type:          store.FinancialEventDeposit,
				AmountKopecks: 100,
				Currency:      "RUB",
				SourceText:    "отложил 1 рубль",
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	goal, err = dataStore.Goals.ByID(ctx, goal.ID, user.Id)
	require.NoError(t, err)
	require.EqualValues(t, eventCount*100, goal.CurrentAmountKopecks)
	require.Equal(t, "completed", goal.Status)
}
