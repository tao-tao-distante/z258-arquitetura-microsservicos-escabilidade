package correlation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFromContextMissing(t *testing.T) {
	require.Equal(t, "", FromContext(context.Background()))
}

func TestNewContextRoundTrip(t *testing.T) {
	ctx := NewContext(context.Background(), "abc123")
	require.Equal(t, "abc123", FromContext(ctx))
}

func TestNewIDFormat(t *testing.T) {
	id := NewID()
	_, err := uuid.Parse(id)
	require.NoError(t, err)
}
