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
	parsed, err := uuid.Parse(id)
	require.NoError(t, err)
	require.Equal(t, uuid.Version(4), parsed.Version())
}

func TestValidIDAcceptsOnlyUUIDv4(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "v4", id: "646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d", want: true},
		{name: "v1", id: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		{name: "invalid", id: "not-a-uuid"},
		{name: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ValidID(tt.id))
		})
	}
}
