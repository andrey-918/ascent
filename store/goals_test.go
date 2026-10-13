package store_test

import (
	"ascent/fixtures"
	"ascent/store"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoalStore(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	ctx := context.Background()
	userStore := store.NewUserStore(env.Db)
	user, err := userStore.CreateUser(ctx, "goals@test.com", "testingpassword")
	require.NoError(t, err)
	otherUser, err := userStore.CreateUser(ctx, "other-goals@test.com", "testingpassword")
	require.NoError(t, err)

	goalStore := store.NewGoalStore(env.Db)
	goal, err := goalStore.Create(ctx, user.Id, "Emergency fund", 5_000_000)
	require.NoError(t, err)
	require.Equal(t, user.Id, goal.UserID)
	require.Equal(t, "Emergency fund", goal.Name)
	require.EqualValues(t, 5_000_000, goal.TargetAmountKopecks)
	require.Zero(t, goal.CurrentAmountKopecks)
	require.Equal(t, "RUB", goal.Currency)
	require.Equal(t, "active", goal.Status)

	found, err := goalStore.ByID(ctx, goal.ID, user.Id)
	require.NoError(t, err)
	require.Equal(t, goal, found)

	goals, err := goalStore.ListByUser(ctx, user.Id)
	require.NoError(t, err)
	require.Equal(t, []store.Goal{*goal}, goals)

	_, err = goalStore.ByID(ctx, goal.ID, otherUser.Id)
	require.ErrorIs(t, err, store.ErrGoalNotFound)
}
