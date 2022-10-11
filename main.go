package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const localRepoPath = "/etc/yum.repos.d/"
const localVarsPath = "/etc/dnf/vars"

var baseVariables = map[string]string{
	"arch":       "aarch64",
	"basearch":   "aarch64",
	"releasever": "2022.0.20220928",
}

func main() {
	vars, err := readVars(localVarsPath)
	if err != nil {
		panic(err)
	}
	varsReplacer := buildVarsReplacer(baseVariables, vars)

	repos, err := ReadRepositories(localRepoPath, varsReplacer)
	if err != nil {
		panic(err)
	}

	for _, repo := range repos {
		if !repo.Enabled {
			continue
		}

		if err := repo.Dbg(); err != nil {
			panic(err)
		}
	}

	fmt.Println("SUCCESS")
}

func readVars(varsDir string) (map[string]string, error) {
	varsFile, err := os.ReadDir(varsDir)
	if err != nil {
		return nil, err
	}

	vars := make(map[string]string)
	for _, f := range varsFile {
		if f.IsDir() {
			continue
		}

		varName := f.Name()
		value, err := os.ReadFile(filepath.Join(varsDir, varName))
		if err != nil {
			return nil, err
		}

		vars[varName] = strings.TrimSpace(string(value))
	}
	return vars, nil
}

func buildVarsReplacer(varMaps ...map[string]string) *strings.Replacer {
	count := 0
	for _, varMap := range varMaps {
		count += len(varMap)
	}

	pairs := make([]string, 0, count*2)
	for _, varMap := range varMaps {
		for name, value := range varMap {
			pairs = append(pairs, "$"+name, value)
		}
	}

	return strings.NewReplacer(pairs...)
}
