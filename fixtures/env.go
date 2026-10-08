package fixtures

import (
	"ascent/config"
	"ascent/store"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

type TestEnv struct {
	Config *config.Config
	Db     *sql.DB
}

func NewTestEnv(t *testing.T) *TestEnv {
	os.Setenv("ENV", string(config.EnvTest))
	conf, err := config.New()
	require.NoError(t, err)

	db, err := store.NewPostgresDB(conf)
	require.NoError(t, err)

	return &TestEnv{Config: conf, Db: db}
}

func migrationsURL() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..")
	return "file://" + filepath.Join(root, "migrations")
}

func (te *TestEnv) SetupDb(t *testing.T) func(t *testing.T) {
	m, err := migrate.New(migrationsURL(), te.Config.DatabaseUrl())
	require.NoError(t, err)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	_, err = te.Db.Exec("TRUNCATE TABLE users, refresh_tokens RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	return func(t *testing.T) {
		_ = te.Db.Close()
	}
}
