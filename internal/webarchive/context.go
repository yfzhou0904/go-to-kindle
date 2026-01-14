package webarchive

import (
	"context"
	"net/url"
)

type contextKey string

const archiveKey contextKey = "webarchive"

type archiveContext struct {
	baseURL   *url.URL
	resources map[string]Resource
}

func WithArchive(ctx context.Context, baseURL *url.URL, resources map[string]Resource) context.Context {
	return context.WithValue(ctx, archiveKey, archiveContext{
		baseURL:   baseURL,
		resources: resources,
	})
}

func GetArchive(ctx context.Context) (*url.URL, map[string]Resource, bool) {
	value := ctx.Value(archiveKey)
	if value == nil {
		return nil, nil, false
	}
	ac, ok := value.(archiveContext)
	if !ok {
		return nil, nil, false
	}
	if ac.baseURL == nil || ac.resources == nil {
		return nil, nil, false
	}
	return ac.baseURL, ac.resources, true
}
