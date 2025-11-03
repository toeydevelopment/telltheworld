package wid

import (
	"github.com/google/uuid"
	"go.jetify.com/typeid"
)

type WID struct {
}

func New() WID {
	return WID{}
}

func (w WID) Gen(prefix string) string {
	id, err := typeid.WithPrefix(prefix)
	if err != nil {
		return uuid.New().String()
	}
	return id.String()
}
