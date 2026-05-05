package config

import (
	"flag"
	"io"
	"net"
	"net/url"
	"os"

	env "github.com/caarlos0/env/v11"
	"github.com/rs/zerolog/log"
)

const defaultRunAddress = "localhost:8080"

// Configuration содержит настройки сервиса накопительной системы лояльности.
type Configuration struct {
	Server   ServerConfiguration
	Database DatabaseConfiguration
	Accrual  AccrualConfiguration
}

// ServerConfiguration содержит настройки запуска HTTP API.
type ServerConfiguration struct {
	Address string
}

// DatabaseConfiguration содержит настройки подключения к PostgreSQL.
type DatabaseConfiguration struct {
	URI *string
}

// IsConfigured сообщает, что адрес подключения к PostgreSQL задан.
func (c DatabaseConfiguration) IsConfigured() bool {
	return c.URI != nil
}

// AccrualConfiguration содержит адрес системы расчёта начислений.
type AccrualConfiguration struct {
	Address *string
}

// IsConfigured сообщает, что адрес системы расчёта начислений задан.
func (c AccrualConfiguration) IsConfigured() bool {
	return c.Address != nil
}

// LoadConfig загружает конфигурацию из аргументов текущего процесса и
// переменных окружения.
func LoadConfig() (*Configuration, error) {
	return loadConfig(os.Args[1:])
}

func defaultConfig() *Configuration {
	return &Configuration{
		Server: ServerConfiguration{
			Address: defaultRunAddress,
		},
	}
}

func loadConfig(args []string) (*Configuration, error) {
	cfg := defaultConfig()

	cliCfg, err := parseCLIArgs(args)
	if err != nil {
		return nil, err
	}

	applyCLIConfig(cfg, cliCfg)

	envCfg, err := env.ParseAsWithOptions[envConfig](env.Options{})
	if err != nil {
		return nil, err
	}

	applyEnvConfig(cfg, envCfg)

	return cfg, nil
}

type cliConfig struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
}

type envConfig struct {
	RunAddress           *string `env:"RUN_ADDRESS"`
	DatabaseURI          *string `env:"DATABASE_URI"`
	AccrualSystemAddress *string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func parseCLIArgs(args []string) (cliConfig, error) {
	var cfg cliConfig

	fs := flag.NewFlagSet("gophermart", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.RunAddress, "a", "", "server run address")
	fs.StringVar(&cfg.DatabaseURI, "d", "", "database uri")
	fs.StringVar(&cfg.AccrualSystemAddress, "r", "", "accrual system address")

	if err := fs.Parse(args); err != nil {
		return cliConfig{}, err
	}

	return cfg, nil
}

func applyCLIConfig(cfg *Configuration, cliCfg cliConfig) {
	cfg.Server.Address = resolveRunAddress(cfg.Server.Address, cliCfg.RunAddress)
	cfg.Database.URI = resolveDatabaseURI(cliCfg.DatabaseURI)
	cfg.Accrual.Address = resolveAccrualSystemAddress(cliCfg.AccrualSystemAddress)
}

func applyEnvConfig(cfg *Configuration, envCfg envConfig) {
	if envCfg.RunAddress != nil {
		if isValidRunAddress(*envCfg.RunAddress) {
			cfg.Server.Address = *envCfg.RunAddress
			log.Info().Str("RunAddress", *envCfg.RunAddress).Msg("Overriding RunAddress from environment")
		} else {
			log.Warn().Str("RunAddress", *envCfg.RunAddress).Msg("Invalid RunAddress from environment, using previous value")
		}
	}

	if envCfg.DatabaseURI != nil {
		if *envCfg.DatabaseURI != "" {
			cfg.Database.URI = envCfg.DatabaseURI
			log.Info().Str("DatabaseURI", *envCfg.DatabaseURI).Msg("Overriding DatabaseURI from environment")
		} else {
			log.Warn().Msg("Empty DatabaseURI from environment, using previous value")
		}
	}

	if envCfg.AccrualSystemAddress != nil {
		if isValidURL(*envCfg.AccrualSystemAddress) {
			cfg.Accrual.Address = envCfg.AccrualSystemAddress
			log.Info().Str("AccrualSystemAddress", *envCfg.AccrualSystemAddress).Msg("Overriding AccrualSystemAddress from environment")
		} else {
			log.Warn().Str("AccrualSystemAddress", *envCfg.AccrualSystemAddress).Msg("Invalid AccrualSystemAddress from environment, using previous value")
		}
	}
}

func resolveRunAddress(defaultValue, cliValue string) string {
	if cliValue != "" {
		if isValidRunAddress(cliValue) {
			log.Info().Str("RunAddress", cliValue).Msg("Overriding RunAddress from CLI")
			return cliValue
		}
		log.Warn().Str("RunAddress", cliValue).Msg("Invalid CLI RunAddress, using default value")
	}

	log.Info().Str("RunAddress", defaultValue).Msg("Using default RunAddress")
	return defaultValue
}

func resolveDatabaseURI(cliValue string) *string {
	if cliValue != "" {
		log.Info().Str("DatabaseURI", cliValue).Msg("Overriding DatabaseURI from CLI")
		return &cliValue
	}

	log.Info().Str("DatabaseURI", "").Msg("Using default DatabaseURI")
	return nil
}

func resolveAccrualSystemAddress(cliValue string) *string {
	if cliValue != "" {
		if isValidURL(cliValue) {
			log.Info().Str("AccrualSystemAddress", cliValue).Msg("Overriding AccrualSystemAddress from CLI")
			return &cliValue
		}
		log.Warn().Str("AccrualSystemAddress", cliValue).Msg("Invalid CLI AccrualSystemAddress, using default value")
	}

	log.Info().Str("AccrualSystemAddress", "").Msg("Using default AccrualSystemAddress")
	return nil
}

func isValidRunAddress(address string) bool {
	_, err := net.ResolveTCPAddr("tcp", address)
	return err == nil
}

func isValidURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme != "" && u.Host != ""
}
