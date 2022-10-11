package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/paulcacheux/did-not-finish/internal/utils"
	"github.com/paulcacheux/did-not-finish/repo"
)

func main() {
	var repoPath, varsPaths string

	flag.StringVar(&repoPath, "repos", "/etc/yum.repos.d/", "path to repos")
	flag.StringVar(&varsPaths, "vars", "/etc/dnf/vars/,/etc/yum/vars/", "paths to variables")

	flag.Parse()

	builtinVars, err := computeBuiltinVariables()
	if err != nil {
		panic(err)
	}
	fmt.Println(builtinVars)

	varMaps := []map[string]string{builtinVars}
	for _, varDir := range strings.Split(varsPaths, ",") {
		if varDir == "" {
			continue
		}

		vars, err := readVars(varDir)
		if err != nil {
			continue
		}

		if len(vars) != 0 {
			varMaps = append(varMaps, vars)
		}
	}

	varsReplacer := buildVarsReplacer(varMaps...)

	repos, err := repo.ReadFromDir(repoPath, varsReplacer)
	if err != nil {
		panic(err)
	}

	for _, repository := range repos {
		if !repository.Enabled {
			continue
		}

		if err := repository.Dbg(); err != nil {
			panic(err)
		}
	}

	fmt.Println("SUCCESS")
}

func readVars(varsDir string) (map[string]string, error) {
	varsFile, err := os.ReadDir(utils.HostEtcJoin(varsDir))
	if err != nil {
		return nil, err
	}

	vars := make(map[string]string)
	for _, f := range varsFile {
		if f.IsDir() {
			continue
		}

		varName := f.Name()
		value, err := os.ReadFile(utils.HostEtcJoin(varsDir, varName))
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
