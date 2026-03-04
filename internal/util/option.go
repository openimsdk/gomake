package util

import (
	"errors"
)

func ResolveEnvOption[T any](key string) *T {
	value, err := GetEnv[T](key)
	if err == nil {
		return value
	}
	if errors.Is(err, ErrEnvNotSet) {
		return nil
	}
	return nil
}
