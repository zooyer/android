package user

import (
	"testing"
)

func TestGetgrnam(t *testing.T) {
	t.Log(Getpwnam("inet"))
	t.Log(Getpwnam("u0_a48"))
}
