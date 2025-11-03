package wkratos

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kratos/kratos/v2/middleware/selector"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func NewPathInclude(paths ...string) selector.MatchFunc {
	whiteList := make(map[string]struct{})

	for _, v := range paths {
		whiteList[v] = struct{}{}
	}

	return func(_ context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return true
		}
		return false
	}
}

func NewPathExclude(paths ...string) selector.MatchFunc {
	whiteList := make(map[string]struct{})

	for _, v := range paths {
		whiteList[v] = struct{}{}
	}

	return func(_ context.Context, operation string) bool {
		if _, ok := whiteList[operation]; ok {
			return false
		}
		return true
	}
}

func HTTPRequestFromContext(ctx context.Context) (*http.Request, error) {
	t, ok := khttp.RequestFromServerContext(ctx)
	if !ok {
		return nil, errors.New("http request not found")
	}
	return t, nil
}
