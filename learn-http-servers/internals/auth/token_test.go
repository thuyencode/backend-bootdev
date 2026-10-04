package auth

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

func TestUnit_MakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "Shhh this is a secret"
	expiresIn := time.Minute * 15

	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("want no error, have: %q", err)
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(_ *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		t.Fatalf("want no error, have: %q", err)
	}

	cases := []struct {
		name     string
		expected any
	}{{
		name:     "issuer",
		expected: "chirpy-access",
	}, {
		name:     "issuedAt",
		expected: jwt.NewNumericDate(time.Now()),
	}, {
		name:     "expiresAt",
		expected: jwt.NewNumericDate(time.Now().Add(expiresIn)),
	}, {
		name:     "subject",
		expected: userID.String(),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual any
			var err error

			switch c.name {
			case "issuer":
				actual, err = token.Claims.GetIssuer()
			case "issuedAt":
				actual, err = token.Claims.GetIssuedAt()
			case "expiresAt":
				actual, err = token.Claims.GetExpirationTime()
			case "subject":
				actual, err = token.Claims.GetSubject()
			}

			if err != nil {
				t.Errorf("want no error, have: %q", err)
			}

			if !reflect.DeepEqual(actual, c.expected) {
				t.Errorf("want %q, have %q", c.expected, actual)
			}
		})
	}
}

func TestUnit_ValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "Shhh this is a secret"
	expiresIn := time.Second * 3

	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("want no error, have: %q", err)
	}

	t.Run("subject", func(t *testing.T) {
		actual, err := ValidateJWT(tokenString, tokenSecret)
		if err != nil {
			t.Errorf("want no error, have: %q", err)
		}
		if !reflect.DeepEqual(actual, userID) {
			t.Errorf("want %q, have %q", userID, actual)
		}
	})

	t.Run("should reject expired token(s)", func(t *testing.T) {
		time.Sleep(time.Second * 4)
		actual, err := ValidateJWT(tokenString, tokenSecret)
		expected := uuid.Nil()
		if err == nil {
			t.Errorf("want error to not be nil")
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run("should reject wrong secret", func(t *testing.T) {
		actual, err := ValidateJWT(tokenString, "This is not the secret we want")
		expected := uuid.Nil()
		if err == nil {
			t.Errorf("want error to not be nil")
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})
}

func TestUnit_GetBearerToken(t *testing.T) {
	t.Run("should return token string for valid request", func(t *testing.T) {
		userID := uuid.New()
		tokenSecret := "Shhh this is a secret"
		expiresIn := time.Minute * 15

		expected, err := MakeJWT(userID, tokenSecret, expiresIn)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", expected))

		actual, err := GetBearerToken(req.Header)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run(`should return error for request with no "Authorization" header`, func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		actual, err := GetBearerToken(req.Header)
		if err == nil {
			t.Fatalf("want error to not be nil")
		}

		if !errors.Is(err, ErrAuthorizationHeaderEmpty) {
			t.Errorf("want %q, have %q", ErrAuthorizationHeaderEmpty, err)
		}

		expected := ""
		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run(`should return error for request with no "Bearer"`, func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		req.Header.Add("Authorization", "Bear")

		actual, err := GetBearerToken(req.Header)
		if err == nil {
			t.Fatalf("want error to not be nil")
		}

		if !errors.Is(err, ErrNoBearerInAuthorizationHeader) {
			t.Errorf("want %q, have %q", ErrNoBearerInAuthorizationHeader, err)
		}

		expected := ""
		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run(`should return error for request with no token string`, func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		req.Header.Add("Authorization", "Bearer ")

		actual, err := GetBearerToken(req.Header)
		if err == nil {
			t.Fatalf("want error to not be nil")
		}

		if !errors.Is(err, ErrNoTokenAfterBearer) {
			t.Errorf("want %q, have %q", ErrNoTokenAfterBearer, err)
		}

		expected := ""
		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})
}

func TestUnit_GetAPIKey(t *testing.T) {
	t.Run("should return token string for valid request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		expected := "This is an api key"
		req.Header.Add("Authorization", fmt.Sprintf("ApiKey %s", expected))

		actual, err := GetAPIKey(req.Header)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run(`should return error for request with no "Authorization" header`, func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		actual, err := GetAPIKey(req.Header)
		if err == nil {
			t.Fatalf("want error to not be nil")
		}

		expectedErr := ErrAuthorizationHeaderEmpty
		if !errors.Is(err, expectedErr) {
			t.Errorf("want %q, have %q", expectedErr, err)
		}

		expected := ""
		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run(`should return error for request with no "ApiKey"`, func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		req.Header.Add("Authorization", "Api")

		actual, err := GetAPIKey(req.Header)
		if err == nil {
			t.Fatalf("want error to not be nil")
		}

		expectedErr := ErrNoApiKeyInAuthorizationHeader
		if !errors.Is(err, expectedErr) {
			t.Errorf("want %q, have %q", expectedErr, err)
		}

		expected := ""
		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})

	t.Run(`should return error for request with no token string`, func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("want no error, have: %q", err)
		}

		req.Header.Add("Authorization", "ApiKey ")

		actual, err := GetAPIKey(req.Header)
		if err == nil {
			t.Fatalf("want error to not be nil")
		}

		expectedErr := ErrNoTokenAfterApiKey
		if !errors.Is(err, expectedErr) {
			t.Errorf("want %q, have %q", expectedErr, err)
		}

		expected := ""
		if actual != expected {
			t.Errorf("want %q, have %q", expected, actual)
		}
	})
}
