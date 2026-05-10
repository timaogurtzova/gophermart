package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/repository"
)

func TestNewUserRepository(t *testing.T) {
	store, err := repository.NewUserRepository(nil)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.ErrorIs(t, err, repository.ErrDatabaseNotConfigured)
}

func TestNewOrderRepository(t *testing.T) {
	store, err := repository.NewOrderRepository(nil)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.ErrorIs(t, err, repository.ErrDatabaseNotConfigured)
}

func TestNewBalanceRepository(t *testing.T) {
	store, err := repository.NewBalanceRepository(nil)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.ErrorIs(t, err, repository.ErrDatabaseNotConfigured)
}
