//go:build !linux

package main

func computeBuiltinVariables() (map[string]string, error) {
	return map[string]string{
		"arch":       "aarch64",
		"basearch":   "aarch64",
		"releasever": "2022.0.20220928",
	}, nil
}
