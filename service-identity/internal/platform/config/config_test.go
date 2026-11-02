package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrJWTSecretRequired))
}

func TestLoadDefaultsWithRequiredSecret(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "secret")
	c, err := Load()
	require.NoError(t, err)
	require.Equal(t, "8080", c.Port)
	require.Contains(t, c.DatabaseURL, "identity")
	require.Equal(t, "secret", c.JWTSecret)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("DATABASE_URL", "url")
	t.Setenv("JWT_SECRET", "secret")
	c, err := Load()
	require.NoError(t, err)
	require.Equal(t, "9999", c.Port)
	require.Equal(t, "url", c.DatabaseURL)
	require.Equal(t, "secret", c.JWTSecret)
}
