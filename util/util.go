package util

import (
	"context"
	"log"
	"os"
	"path/filepath"
)

// Debug context utilities
type ctxKeyDebug struct{}

func WithDebug(ctx context.Context, debug bool) context.Context {
	return context.WithValue(ctx, ctxKeyDebug{}, debug)
}

func Debug(ctx context.Context) bool {
	v := ctx.Value(ctxKeyDebug{})
	b, _ := v.(bool)
	return b
}

// BaseDir returns the base directory for go-to-kindle data
func BaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(home, ".go-to-kindle")
}