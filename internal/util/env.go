package util

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var ErrEnvNotSet = errors.New("environment variable not set")
var ErrUnsupportedEnvType = errors.New("unsupported env type")

func GetEnv[T any](key string) (*T, error) {
	var zero T
	raw, ok := os.LookupEnv(key)
	if !ok {
		return nil, ErrEnvNotSet
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrEnvNotSet
	}

	switch any(zero).(type) {
	case string:
		value := any(raw).(T)
		return &value, nil
	case bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s=%q as bool: %w", key, raw, err)
		}
		resolved := any(value).(T)
		return &resolved, nil
	case int:
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s=%q as int: %w", key, raw, err)
		}
		resolved := any(value).(T)
		return &resolved, nil
	case uint64:
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse %s=%q as uint64: %w", key, raw, err)
		}
		resolved := any(value).(T)
		return &resolved, nil
	case []string:
		values := strings.Fields(raw)
		if len(values) == 0 {
			return nil, ErrEnvNotSet
		}
		resolved := any(values).(T)
		return &resolved, nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedEnvType, zero)
	}
}

func SetEnvs(envMap map[string]string) (func(), error) {
	oldEnv := make(map[string]string)
	restore := func() {
		for k, v := range oldEnv {
			_ = os.Setenv(k, v)
		}
		for k := range envMap {
			if _, existed := oldEnv[k]; !existed {
				_ = os.Unsetenv(k)
			}
		}
	}
	for k, v := range envMap {
		old, exist := os.LookupEnv(k)
		if exist {
			oldEnv[k] = old
		}
		if err := os.Setenv(k, v); err != nil {
			restore()
			return nil, err
		}
	}
	return restore, nil
}
