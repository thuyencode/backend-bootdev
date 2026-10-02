package auth

import (
	"testing"
	"unsafe"
)

func TestUnit_MakeRefreshToken(t *testing.T) {
	token := MakeRefreshToken()

	if token == "" {
		t.Error("want refresh token to not be empty")
	}

	actualSize := unsafe.Sizeof(token)
	expectedSize := 16
	if actualSize != uintptr(expectedSize) {
		t.Errorf("want refresh token to be a %d bytes string, have: %d", expectedSize, actualSize)
	}
}
