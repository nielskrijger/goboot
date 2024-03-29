package goboot

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// AppEnv contains all application-scoped variables.
type AppEnv struct {
	Config   *viper.Viper
	Log      zerolog.Logger
	ConfDir  string
	Services []AppService
}

// NewAppEnv creates an AppEnv by loading configuration settings.
//
// Panics if configuration failed to load.
func NewAppEnv(confDir string, env string) *AppEnv {
	logger := newLogger()
	logger.Info().Str("env", env).Msgf("starting server")

	cfg, err := LoadConfig(logger, confDir, env)
	if err != nil {
		log.Panic().Err(err).Msgf("loading app configs: %s", err.Error())
	}

	// Set log settings after we've loaded the config files
	if level := cfg.GetString("log.level"); level != "" {
		SetGlobalLogLevel(level)
	}

	if humanize := cfg.GetString("log.human"); humanize == "true" {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	return &AppEnv{
		ConfDir:  confDir,
		Config:   cfg,
		Log:      logger,
		Services: make([]AppService, 0),
	}
}

func (ctx *AppEnv) AddService(service AppService) {
	ctx.Services = append(ctx.Services, service)
}

// newLogger configures a new zerolog logger.
//
// By default, returns a production logger. For debugging set the following values:
//
//   - LOG_LEVEL=debug
//   - LOG_HUMAN=true
//
// The LOG_* env vars can be defined in config files using "log.level" and "log.human"
// but will only take effect after the config files are loaded while LOG_* will takes
// immediate effect.
func newLogger() zerolog.Logger {
	// use env var instead of config because no config is available at startup
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if level, ok := os.LookupEnv("LOG_LEVEL"); ok {
		SetGlobalLogLevel(level)
	}

	human, ok := os.LookupEnv("LOG_HUMAN")

	if ok && (human == "true") {
		return log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	return zerolog.New(os.Stdout)
}

// SetGlobalLogLevel updates the log level, panics if log level is unknown.
func SetGlobalLogLevel(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Panic().Err(err).Msgf("setting log level: %s", err.Error())
	}

	zerolog.SetGlobalLevel(lvl)
}

// Configure sets up service settings.
func (ctx *AppEnv) Configure() {
	ctx.Log.Info().Msg("starting configuring app services")

	for _, service := range ctx.Services {
		if err := service.Configure(ctx); err != nil {
			ctx.Log.Panic().Err(err).Msgf("failed to configure service %s", service.Name())
		}
	}

	ctx.Log.Info().Msg("finished configuring app services")
}

// Init runs all app service initialization.
func (ctx *AppEnv) Init() {
	ctx.Log.Info().Msg("starting app services init")

	for _, service := range ctx.Services {
		if err := service.Init(); err != nil {
			ctx.Log.Panic().Err(err).Msgf("failed to initialize service %s", service.Name())
		}
	}

	ctx.Log.Info().Msg("finished app services init")
}

// Close cleans up any resources held by any app services.
func (ctx *AppEnv) Close() {
	ctx.Log.Info().Msg("start closing app services")

	for _, service := range ctx.Services {
		if err := service.Close(); err != nil {
			ctx.Log.Error().Err(err).Msgf("failed to gracefully close service %s", service.Name())
		}
	}

	ctx.Log.Info().Msg("finished closing app services")
}
