package wboot

import (
	"os"
	"strings"
)

func envOr(key string, d string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return d
}
