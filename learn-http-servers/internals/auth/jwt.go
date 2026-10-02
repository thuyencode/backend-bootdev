package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrFailedToClaimToken            = errors.New("failed to claim token")
	ErrAuthorizationHeaderEmpty      = errors.New(`the "Authorization" header is empty`)
	ErrNoBearerInAuthorizationHeader = errors.New(
		`"Bearer" is absent in the "Authorization" header`,
	)
	ErrNoTokenAfterBearer = errors.New(
		`the token is absent after "Bearer" in the "Authorization" header`,
	)
)

const issuer = "chirpy-access"

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(_ *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.Nil(), err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
		return uuid.Parse(claims.Subject)
	}

	return uuid.Nil(), ErrFailedToClaimToken
}

func GetBearerToken(headers http.Header) (string, error) {
	authorization := headers.Get("Authorization")
	if authorization == "" {
		return "", ErrAuthorizationHeaderEmpty
	}

	if !strings.Contains(authorization, "Bearer ") {
		return "", ErrNoBearerInAuthorizationHeader
	}

	tokenString := strings.TrimPrefix(authorization, "Bearer ")
	if tokenString == "" {
		return "", ErrNoTokenAfterBearer
	}

	return tokenString, nil
}
