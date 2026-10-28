package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Issuer e Audience identificam o emissor e o público esperados nos tokens JWT.
const (
	Issuer   = "service-identity"
	Audience = "service-task"
)

// ErrInvalidClaims indica que as claims do token são inválidas ou incompletas.
var ErrInvalidClaims = errors.New("invalid token claims")

// Claims carrega a identidade e o e-mail de um token de acesso.
type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Validate informa se as claims contêm um identificador de usuário utilizável.
func (c Claims) Validate() error {
	if c.UserID <= 0 {
		return ErrInvalidClaims
	}
	return nil
}

// HashPassword retorna o hash bcrypt de password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword informa se password corresponde ao hash bcrypt.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken assina um token de acesso HS256 para userID e email com o TTL informado.
func GenerateToken(secret string, userID int64, email string, ttl time.Duration) (string, error) {
	if userID <= 0 {
		return "", ErrInvalidClaims
	}
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken valida um token HS256, incluindo emissor, audiência e expiração.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ParseAccessToken valida um token e exige as claims user_id e email.
func ParseAccessToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	if claims.UserID <= 0 || strings.TrimSpace(claims.Email) == "" {
		return nil, errors.New("required token claims are missing")
	}
	return claims, nil
}

// TokenFromHeader extrai o token de um header Authorization no formato Bearer.
func TokenFromHeader(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("missing bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errors.New("empty token")
	}
	return token, nil
}

type ctxKey string

const claimsKey ctxKey = "claims"

// WithClaims devolve um contexto com as claims autenticadas.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// FromContext devolve as claims autenticadas do contexto, ou nil.
func FromContext(ctx context.Context) *Claims {
	if v := ctx.Value(claimsKey); v != nil {
		if c, ok := v.(*Claims); ok {
			return c
		}
	}
	return nil
}

// Middleware exige um Bearer token válido e injeta as claims no contexto.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := TokenFromHeader(r.Header.Get("Authorization"))
			if err != nil {
				writeUnauthorized(w)
				return
			}
			claims, err := ParseAccessToken(secret, token)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}
