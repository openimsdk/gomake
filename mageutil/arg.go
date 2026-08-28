package mageutil

import (
	"strings"

	"github.com/openimsdk/tools/utils/datautil"
)

func ParseArgList(arg *string) []string {
	if arg == nil || *arg == "" {
		return nil
	}

	return datautil.Filter(strings.Split(*arg, ","), func(part string) (string, bool) {
		part = strings.TrimSpace(part)
		return part, part != ""
	})
}
