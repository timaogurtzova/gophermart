package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timaogurtzova/gophermart/internal/model"
	"github.com/timaogurtzova/gophermart/internal/repository"
	"github.com/timaogurtzova/gophermart/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthServiceRegisterHashesPassword(t *testing.T) {
	users := &fakeUserRepository{
		create: func(_ context.Context, login, passwordHash string) (model.User, error) {
			assert.Equal(t, "user", login)
			assert.NotEqual(t, "password", passwordHash)
			assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("password")))

			return model.User{ID: 42, Login: login, PasswordHash: passwordHash}, nil
		},
	}
	svc := service.NewAuthService(users)

	user, err := svc.Register(context.Background(), "user", "password")
	require.NoError(t, err)

	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "user", user.Login)
}

func TestAuthServiceRegisterMapsLoginConflict(t *testing.T) {
	users := &fakeUserRepository{
		create: func(_ context.Context, _, _ string) (model.User, error) {
			return model.User{}, repository.ErrUserAlreadyExists
		},
	}
	svc := service.NewAuthService(users)

	_, err := svc.Register(context.Background(), "user", "password")
	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrLoginAlreadyTaken)
}

func TestAuthServiceLogin(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	users := &fakeUserRepository{
		findByLogin: func(_ context.Context, login string) (model.User, error) {
			assert.Equal(t, "user", login)
			return model.User{ID: 42, Login: login, PasswordHash: string(passwordHash)}, nil
		},
	}
	svc := service.NewAuthService(users)

	user, err := svc.Login(context.Background(), "user", "password")
	require.NoError(t, err)

	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "user", user.Login)
}

func TestAuthServiceLoginReturnsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name        string
		repository  repository.UserRepository
		password    string
		wantErrText string
	}{
		{
			name: "user not found",
			repository: &fakeUserRepository{
				findByLogin: func(_ context.Context, _ string) (model.User, error) {
					return model.User{}, repository.ErrUserNotFound
				},
			},
			password:    "password",
			wantErrText: service.ErrInvalidCredentials.Error(),
		},
		{
			name: "wrong password",
			repository: &fakeUserRepository{
				findByLogin: func(_ context.Context, _ string) (model.User, error) {
					passwordHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
					require.NoError(t, err)
					return model.User{ID: 42, Login: "user", PasswordHash: string(passwordHash)}, nil
				},
			},
			password:    "wrong-password",
			wantErrText: service.ErrInvalidCredentials.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewAuthService(tt.repository)

			_, err := svc.Login(context.Background(), "user", tt.password)
			require.Error(t, err)
			assert.ErrorIs(t, err, service.ErrInvalidCredentials)
			assert.Equal(t, tt.wantErrText, err.Error())
		})
	}
}

type fakeUserRepository struct {
	create      func(ctx context.Context, login, passwordHash string) (model.User, error)
	findByLogin func(ctx context.Context, login string) (model.User, error)
}

func (r *fakeUserRepository) Create(ctx context.Context, login, passwordHash string) (model.User, error) {
	if r.create == nil {
		return model.User{}, nil
	}

	return r.create(ctx, login, passwordHash)
}

func (r *fakeUserRepository) FindByLogin(ctx context.Context, login string) (model.User, error) {
	if r.findByLogin == nil {
		return model.User{}, nil
	}

	return r.findByLogin(ctx, login)
}
