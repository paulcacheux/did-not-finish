package main

import (
	"os"
	"path/filepath"
	"strings"
)

const etcPrefix = "/etc"

func hostEtcJoin(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	first := parts[0]
	hostEtc := os.Getenv("HOST_ETC")
	if hostEtc == "" || !strings.HasPrefix(first, etcPrefix) {
		return filepath.Join(parts...)
	}

	first = strings.TrimPrefix(first, etcPrefix)
	newParts := make([]string, len(parts)+1)
	newParts[0] = hostEtc
	newParts[1] = first
	if len(parts) > 1 {
		copy(newParts[2:], parts[1:])
	}
	return filepath.Join(newParts...)
}
