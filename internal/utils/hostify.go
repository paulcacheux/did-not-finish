package utils

import (
	"os"
	"path/filepath"
	"strings"
)

const etcPrefix = "/etc"

func rawHostJoin(envName, defaultValue string, parts ...string) string {
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

func HostEtcJoin(parts ...string) string {
	return rawHostJoin("HOST_ETC", "/etc", parts...)
}

func HostVarJoin(parts ...string) string {
	return rawHostJoin("HOST_VAR", "/var", parts...)
}
