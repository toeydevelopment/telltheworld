package wkratos

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware/selector"
)

func NewOperationInclude(operations ...string) selector.MatchFunc {
	whiteList := make(map[string]struct{})

	for _, v := range operations {
		whiteList[v] = struct{}{}
	}

	return func(_ context.Context, operation string) bool {

		if _, ok := whiteList[operation]; ok {
			return true
		}

		return false
	}
}

func NewOperationExclude(operations ...string) selector.MatchFunc {

	whiteList := make(map[string]struct{})

	for _, v := range operations {
		whiteList[v] = struct{}{}
	}

	return func(_ context.Context, operation string) bool {

		if _, ok := whiteList[operation]; ok {
			return false
		}

		return true
	}
}
