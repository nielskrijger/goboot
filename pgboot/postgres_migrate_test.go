package pgboot_test

import (
	"testing"

	"github.com/nielskrijger/goboot"
	"github.com/nielskrijger/goboot/pgboot"
	"github.com/nielskrijger/goboot/test"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type Record struct {
	ID   int
	Name string
}

func TestPostgresMigrate_Success(t *testing.T) {
	s := &pgboot.Postgres{MigrationsDir: "./testdata/migrations"}
	env := goboot.NewAppEnv("./testdata", "valid")
	assert.Nil(t, s.Configure(env))
	_, _ = s.DB.Exec("DROP TABLE IF EXISTS test_table")
	_, _ = s.DB.Exec("DROP TABLE IF EXISTS schema_migrations")
	assert.Nil(t, s.Init())

	var records []Record
	err := s.DB.Select(&records, "SELECT * FROM test_table")
	assert.Nil(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, "First record", records[0].Name)
	assert.Equal(t, "Second record", records[1].Name)
}

func TestPostgresMigrate_SkipMigrationsWhenDirEmpty(t *testing.T) {
	log := &test.Logger{}
	s := &pgboot.Postgres{}
	env := goboot.NewAppEnv("./testdata", "valid")
	env.Log = zerolog.New(log)
	assert.Nil(t, s.Configure(env))
	assert.Nil(t, s.Init())

	assert.Equal(t, "skipping db migrations; no migrations directory set", log.LastLine()["message"])
}
