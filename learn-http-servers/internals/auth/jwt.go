package auth

import (
	"errors"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

var ErrFailedToClaimToken = errors.New("failed to claim token")

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
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
