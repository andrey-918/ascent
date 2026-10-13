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
	Config   *config.Config
	Db       *sql.DB
	testLock *sql.Conn
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
	ctx := t.Context()
	lock, err := te.Db.Conn(ctx)
	require.NoError(t, err)
	_, err = lock.ExecContext(ctx, `SELECT pg_advisory_lock(1095976259, 1)`)
	require.NoError(t, err)
	te.testLock = lock

	m, err := migrate.New(migrationsURL(), te.Config.DatabaseUrl())
	require.NoError(t, err)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	_, err = te.Db.Exec("TRUNCATE TABLE users, refresh_tokens RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	return func(t *testing.T) {
		_, err := te.testLock.ExecContext(t.Context(), `SELECT pg_advisory_unlock(1095976259, 1)`)
		require.NoError(t, err)
		require.NoError(t, te.testLock.Close())
		_ = te.Db.Close()
	}
}
