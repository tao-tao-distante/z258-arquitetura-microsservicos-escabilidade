package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func makeAccessToken(t *testing.T, secret string, exp time.Duration) string {
	t.Helper()
	claims := Claims{UserID: 5, Email: "a@b.com", RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
	}}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("s3cret")
	require.NoError(t, err)
	require.NotEqual(t, "s3cret", hash)
	require.True(t, CheckPassword(hash, "s3cret"))
	require.False(t, CheckPassword(hash, "wrong"))
}

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken("secret", 42, "a@b.com", time.Hour)
	require.NoError(t, err)

	claims, err := ParseToken("secret", token)
	require.NoError(t, err)
	require.Equal(t, int64(42), claims.UserID)
	require.Equal(t, "a@b.com", claims.Email)
	require.Equal(t, Issuer, claims.Issuer)
	require.Contains(t, claims.Audience, Audience)
}

func TestParseTokenRejectsNonHS256HMAC(t *testing.T) {
	claims := Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Audience: jwt.ClaimStrings{Audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("secret"))
	require.NoError(t, err)
	_, err = ParseToken("secret", token)
	require.Error(t, err)
}

func TestParseTokenRequiresIdentityClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims Claims
	}{
		{name: "issuer", claims: Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{Issuer: "other", Audience: jwt.ClaimStrings{Audience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
		{name: "audience", claims: Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{Issuer: Issuer, Audience: jwt.ClaimStrings{"other"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
		{name: "user id", claims: Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: Issuer, Audience: jwt.ClaimStrings{Audience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tt.claims).SignedString([]byte("secret"))
			require.NoError(t, err)
			_, err = ParseToken("secret", token)
			require.Error(t, err)
		})
	}
}

func TestGenerateTokenRejectsInvalidUserID(t *testing.T) {
	_, err := GenerateToken("secret", 0, "a@b.com", time.Hour)
	require.True(t, errors.Is(err, ErrInvalidClaims))
}

func TestParseTokenWrongSecret(t *testing.T) {
	token, err := GenerateToken("secret", 1, "x@y.com", time.Hour)
	require.NoError(t, err)
	_, err = ParseToken("other", token)
	require.Error(t, err)
}

func TestParseTokenExpired(t *testing.T) {
	token, err := GenerateToken("secret", 1, "x@y.com", -time.Hour)
	require.NoError(t, err)
	_, err = ParseToken("secret", token)
	require.Error(t, err)
}

func TestParseTokenMalformed(t *testing.T) {
	_, err := ParseToken("secret", "not-a-jwt")
	require.Error(t, err)
}

func TestHashPasswordTooLong(t *testing.T) {
	long := make([]byte, 100)
	for i := range long {
		long[i] = 'a'
	}
	_, err := HashPassword(string(long))
	require.Error(t, err)
}

func TestParseAccessToken(t *testing.T) {
	c, err := ParseAccessToken("secret", makeAccessToken(t, "secret", time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(5), c.UserID)
}

func TestParseAccessTokenBadSecret(t *testing.T) {
	_, err := ParseAccessToken("other", makeAccessToken(t, "secret", time.Hour))
	require.Error(t, err)
}

func TestParseAccessTokenExpired(t *testing.T) {
	_, err := ParseAccessToken("secret", makeAccessToken(t, "secret", -time.Hour))
	require.Error(t, err)
}

func TestParseAccessTokenMalformed(t *testing.T) {
	_, err := ParseAccessToken("secret", "bad")
	require.Error(t, err)
}

func TestTokenFromHeader(t *testing.T) {
	tok, err := TokenFromHeader("Bearer abc")
	require.NoError(t, err)
	require.Equal(t, "abc", tok)
}

func TestTokenFromHeaderInvalid(t *testing.T) {
	_, err := TokenFromHeader("abc")
	require.Error(t, err)
	_, err = TokenFromHeader("Bearer ")
	require.Error(t, err)
}

func TestMiddlewareRejectsMissing(t *testing.T) {
	h := Middleware("secret", func(*http.Request) {})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareRejectsInvalid(t *testing.T) {
	h := Middleware("secret", func(*http.Request) {})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareReportsUnauthorizedRequest(t *testing.T) {
	called := 0
	h := Middleware("secret", func(*http.Request) { called++ })(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, 1, called)
}

func TestMiddlewareAccepts(t *testing.T) {
	var got *Claims
	h := Middleware("secret", func(*http.Request) {})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+makeAccessToken(t, "secret", time.Hour))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, got)
	require.Equal(t, int64(5), got.UserID)
}

func TestFromContextMissing(t *testing.T) {
	require.Nil(t, FromContext(context.Background()))
}

func TestWithClaimsRoundTrip(t *testing.T) {
	ctx := WithClaims(context.Background(), &Claims{UserID: 9})
	require.Equal(t, int64(9), FromContext(ctx).UserID)
}

func TestParseAccessTokenNoneAlgorithm(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: 1})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = ParseAccessToken("secret", s)
	require.Error(t, err)
}

func TestParseAccessTokenRejectsHS512(t *testing.T) {
	claims := Claims{UserID: 1, Email: "a@b.com", RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("secret"))
	require.NoError(t, err)
	_, err = ParseAccessToken("secret", s)
	require.Error(t, err)
}

func TestParseAccessTokenRequiresStrictClaims(t *testing.T) {
	tests := []Claims{
		{UserID: 1, Email: "a@b.com"},
		{Email: "a@b.com", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
		{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
	}
	for _, claims := range tests {
		s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
		require.NoError(t, err)
		_, err = ParseAccessToken("secret", s)
		require.Error(t, err)
	}
}

func TestMiddlewareUnauthorizedIsJSON(t *testing.T) {
	h := Middleware("secret", func(*http.Request) {})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.JSONEq(t, `{"error":"unauthorized"}`, rec.Body.String())
}
