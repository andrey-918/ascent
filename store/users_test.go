package store

import (
	"ascent/config"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

func migrationsURL() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..")
	return "file://" + filepath.Join(root, "migrations")
}

func TestUserStore(t *testing.T) {
	os.Setenv("ENV", string(config.EnvTest))
	conf, err := config.New()
	require.NoError(t, err)

	db, err := NewPostgresDB(conf)
	require.NoError(t, err)
	defer db.Close()

	m, err := migrate.New(migrationsURL(), conf.DatabaseUrl())
	require.NoError(t, err)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	userStore := NewUserStore(db)
	user, err := userStore.CreateUser(context.Background(), "test@test.com", "testingpassword")
	require.NoError(t, err)

	require.Equal(t, "test@test.com", user.Email)
	require.NoError(t, user.ComparePassword("testingpassword"))
}