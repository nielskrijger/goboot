package goboot

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-pg/pg"
	"github.com/rs/zerolog"
)

const (
	defaultPostgresConnectMaxRetries    = 5
	defaultPostgresConnectRetryDuration = 5 * time.Second
)

var (
	errMissingPostgresConfig = errors.New("missing postgres configuration")
	errMissingPostgresDSN    = errors.New("config \"postgres.dsn\" is required")
)

type PostgresConfig struct {
	// DSN contains hostname:port, e.g. localhost:6379
	DSN string `yaml:"dsn"`

	// Number of seconds before first connect attempt times out.
	ConnectTimeout int `yaml:"connectTimeout"`

	// Number of retries upon initial connect. Default is 5 times. Set -1 to disable
	ConnectMaxRetries int `yaml:"connectMaxRetries"`

	// Time between retries for initial connect attempts. Default is 5 seconds.
	ConnectRetryDuration time.Duration `yaml:"connectRetryDuration"`
}

// Postgres implements the AppService interface.
type Postgres struct {
	MigrationsDir string // relative path to migrations directory, leave empty when no migrations

	DB *pg.DB

	config  *PostgresConfig
	log     zerolog.Logger
	confDir string
}

type dbLogger struct {
	log zerolog.Logger
}

func (d *dbLogger) BeforeQuery(q *pg.QueryEvent) {}

func (d *dbLogger) AfterQuery(q *pg.QueryEvent) {
	str, err := q.FormattedQuery()
	if err != nil {
		d.log.Error().Err(err).Msg("error retrieving query")
	} else {
		d.log.Debug().Msg(str)
	}
}

type healtcheckResult struct {
	Result int
}

func (s *Postgres) Name() string {
	return "postgres"
}

// Configure connects to postgres and logs connection info for
// debugging connectivity issues.
func (s *Postgres) Configure(ctx *AppEnv) error {
	s.log = ctx.Log
	s.confDir = ctx.ConfDir

	// unmarshal config and set defaults
	s.config = &PostgresConfig{}

	if !ctx.Config.InConfig("postgres") {
		return errMissingPostgresConfig
	}

	if !ctx.Config.IsSet("postgres.dsn") {
		return errMissingPostgresDSN
	}

	if err := ctx.Config.Sub("postgres").Unmarshal(s.config); err != nil {
		return fmt.Errorf("parsing postgres configuration: %w", err)
	}

	if s.config.ConnectMaxRetries == 0 {
		s.config.ConnectMaxRetries = defaultPostgresConnectMaxRetries
	}

	if s.config.ConnectRetryDuration == 0*time.Second {
		s.config.ConnectRetryDuration = defaultPostgresConnectRetryDuration
	}

	// check if we can connect to PostgreSQL
	if err := s.testConnectivity(); err != nil {
		return err
	}

	// print SQL queries when debug logging is on
	if ctx.Log.Debug().Enabled() {
		s.DB.AddQueryHook(&dbLogger{log: s.log})
	}

	return nil
}

func (s *Postgres) testConnectivity() error {
	// parse url for logging purposes
	logURL, err := url.Parse(s.config.DSN)
	if err != nil {
		return fmt.Errorf("invalid postgres dsn: %w", err)
	}

	logURL.User = url.UserPassword(logURL.User.Username(), "REDACTED")
	s.log.Info().Msgf("connecting to %s", logURL.String())

	// parse
	pgOptions, err := pg.ParseURL(s.config.DSN)
	if err != nil {
		return fmt.Errorf("could not parse postgres DSN: %w", err)
	}

	pgOptions.DialTimeout = time.Duration(s.config.ConnectTimeout) * time.Second

	for retries := 1; ; retries++ {
		s.DB = pg.Connect(pgOptions)

		// test connection
		if _, err := s.DB.Query(&healtcheckResult{}, "SELECT 1 AS result"); err != nil {
			if retries < s.config.ConnectMaxRetries {
				s.log.
					Warn().
					Err(err).
					Str("url", logURL.String()).
					Msgf("failed to connect to postgres, retrying in %s", s.config.ConnectRetryDuration)
			} else {
				return fmt.Errorf(
					"failed to connect to postgres %q after %d retries: %w",
					logURL.String(),
					s.config.ConnectMaxRetries,
					err,
				)
			}

			time.Sleep(s.config.ConnectRetryDuration)
		} else {
			s.log.Info().Msg("successfully connected to postgres")

			break
		}
	}

	return nil
}

func (s *Postgres) Init() error {
	u, err := url.Parse(s.config.DSN)
	if err != nil {
		return fmt.Errorf("invalid postgres dsn: %w", err)
	}

	q := u.Query()
	q.Set("connect_timeout", strconv.Itoa(s.config.ConnectTimeout))
	u.RawQuery = q.Encode()

	if s.MigrationsDir == "" {
		s.log.Info().Msg("skipping db migrations; no migrations directory set")
	} else if err := s.Migrate(u.String(), s.MigrationsDir); err != nil {
		return fmt.Errorf("running postgres migrations: %w", err)
	}

	return nil
}

func (s *Postgres) Close() error {
	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("closing %s service: %w", s.Name(), err)
	}

	return nil
}
