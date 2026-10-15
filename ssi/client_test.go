package ssi

import (
	"context"
	"testing"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/auth"
)

func TestAuthExposesPollOptions(t *testing.T) {
	var _ func(*Auth, context.Context, auth.EnsureOptions) (string, error) = (*Auth).EnsureAuthenticatedWith

	cfg := NewConfig("c")
	cfg.LogLevel = "ERROR"
	a := NewAuth(cfg)
	if _, err := a.EnsureAuthenticatedWith(context.Background(), auth.EnsureOptions{}); err == nil {
		t.Fatal("without credentials or token the call must fail, not succeed")
	}
}
