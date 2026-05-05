package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/config"
)

func TestLoadConfigPriority(t *testing.T) {
	tests := []struct {
		имя                      string
		args                     []string
		env                      map[string]string
		wantRunAddress           string
		wantDatabaseURI          string
		wantDatabaseConfigured   bool
		wantAccrualSystemAddress string
		wantAccrualSystemConfig  bool
	}{
		{
			имя:            "использует значения по умолчанию, когда нет env и флагов",
			wantRunAddress: "localhost:8080",
		},
		{
			имя:                      "использует флаги, когда env отсутствуют",
			args:                     []string{"-a", "localhost:9090", "-d", "postgres://user:password@localhost:5432/gophermart?sslmode=disable", "-r", "http://localhost:8081"},
			wantRunAddress:           "localhost:9090",
			wantDatabaseURI:          "postgres://user:password@localhost:5432/gophermart?sslmode=disable",
			wantDatabaseConfigured:   true,
			wantAccrualSystemAddress: "http://localhost:8081",
			wantAccrualSystemConfig:  true,
		},
		{
			имя: "использует переменные окружения, когда флаги отсутствуют",
			env: map[string]string{
				"RUN_ADDRESS":            "localhost:7070",
				"DATABASE_URI":           "postgres://env:password@localhost:5432/gophermart?sslmode=disable",
				"ACCRUAL_SYSTEM_ADDRESS": "http://localhost:8082",
			},
			wantRunAddress:           "localhost:7070",
			wantDatabaseURI:          "postgres://env:password@localhost:5432/gophermart?sslmode=disable",
			wantDatabaseConfigured:   true,
			wantAccrualSystemAddress: "http://localhost:8082",
			wantAccrualSystemConfig:  true,
		},
		{
			имя:  "переменные окружения имеют приоритет над флагами",
			args: []string{"-a", "localhost:9090", "-d", "postgres://cli:password@localhost:5432/gophermart?sslmode=disable", "-r", "http://localhost:8081"},
			env: map[string]string{
				"RUN_ADDRESS":            "localhost:7070",
				"DATABASE_URI":           "postgres://env:password@localhost:5432/gophermart?sslmode=disable",
				"ACCRUAL_SYSTEM_ADDRESS": "http://localhost:8082",
			},
			wantRunAddress:           "localhost:7070",
			wantDatabaseURI:          "postgres://env:password@localhost:5432/gophermart?sslmode=disable",
			wantDatabaseConfigured:   true,
			wantAccrualSystemAddress: "http://localhost:8082",
			wantAccrualSystemConfig:  true,
		},
		{
			имя:                      "разбирает поддерживаемые cli-флаги в формате через равно",
			args:                     []string{"-a=localhost:6060", "-d=postgres://user:password@localhost:5432/gophermart?sslmode=disable", "-r=http://localhost:8083"},
			wantRunAddress:           "localhost:6060",
			wantDatabaseURI:          "postgres://user:password@localhost:5432/gophermart?sslmode=disable",
			wantDatabaseConfigured:   true,
			wantAccrualSystemAddress: "http://localhost:8083",
			wantAccrualSystemConfig:  true,
		},
		{
			имя:  "пустые значения из окружения не отключают настроенные флаги",
			args: []string{"-a", "localhost:9090", "-d", "postgres://cli:password@localhost:5432/gophermart?sslmode=disable", "-r", "http://localhost:8081"},
			env: map[string]string{
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "",
				"ACCRUAL_SYSTEM_ADDRESS": "",
			},
			wantRunAddress:           "localhost:9090",
			wantDatabaseURI:          "postgres://cli:password@localhost:5432/gophermart?sslmode=disable",
			wantDatabaseConfigured:   true,
			wantAccrualSystemAddress: "http://localhost:8081",
			wantAccrualSystemConfig:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.имя, func(t *testing.T) {
			cfg, err := loadWithState(t, tt.args, tt.env)
			require.NoError(t, err)

			assert.Equal(t, tt.wantRunAddress, cfg.Server.Address)
			assert.Equal(t, tt.wantDatabaseConfigured, cfg.Database.IsConfigured())
			if tt.wantDatabaseURI == "" {
				assert.Nil(t, cfg.Database.URI)
			} else {
				require.NotNil(t, cfg.Database.URI)
				assert.Equal(t, tt.wantDatabaseURI, *cfg.Database.URI)
			}

			assert.Equal(t, tt.wantAccrualSystemConfig, cfg.Accrual.IsConfigured())
			if tt.wantAccrualSystemAddress == "" {
				assert.Nil(t, cfg.Accrual.Address)
			} else {
				require.NotNil(t, cfg.Accrual.Address)
				assert.Equal(t, tt.wantAccrualSystemAddress, *cfg.Accrual.Address)
			}
		})
	}
}

func TestLoadConfigFallsBackWhenEnvironmentValuesAreInvalid(t *testing.T) {
	cfg, err := loadWithState(t,
		[]string{"-a", "localhost:9090", "-r", "http://localhost:8081"},
		map[string]string{
			"RUN_ADDRESS":            "://bad address",
			"ACCRUAL_SYSTEM_ADDRESS": "bad-url",
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.Server.Address)
	require.NotNil(t, cfg.Accrual.Address)
	assert.Equal(t, "http://localhost:8081", *cfg.Accrual.Address)
}

func TestLoadConfigReturnsCLIParseError(t *testing.T) {
	_, err := loadWithState(t, []string{"-unknown"}, nil)
	require.Error(t, err)
}

func loadWithState(t *testing.T, args []string, envVars map[string]string) (*config.Configuration, error) {
	t.Helper()

	oldArgs := os.Args
	os.Args = append([]string{"gophermart"}, args...)
	t.Cleanup(func() {
		os.Args = oldArgs
	})

	for _, key := range []string{
		"RUN_ADDRESS",
		"DATABASE_URI",
		"ACCRUAL_SYSTEM_ADDRESS",
	} {
		previousValue, wasSet := os.LookupEnv(key)

		if value, ok := envVars[key]; ok {
			require.NoError(t, os.Setenv(key, value))
		} else {
			require.NoError(t, os.Unsetenv(key))
		}

		t.Cleanup(func() {
			if wasSet {
				_ = os.Setenv(key, previousValue)
				return
			}
			_ = os.Unsetenv(key)
		})
	}

	return config.LoadConfig()
}
