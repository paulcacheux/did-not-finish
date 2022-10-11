package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/paulcacheux/did-not-finish/backend"
)

func main() {
	var repoPath, varsPaths string

	flag.StringVar(&repoPath, "repos", "/etc/yum.repos.d/", "path to repos")
	flag.StringVar(&varsPaths, "vars", "/etc/dnf/vars/,/etc/yum/vars/", "paths to variables")

	flag.Parse()

	builtinVars, err := backend.ComputeBuiltinVariables()
	if err != nil {
		panic(err)
	}

	b, err := backend.NewBackend(repoPath, strings.Split(varsPaths, ","), builtinVars)
	if err != nil {
		panic(err)
	}

	for _, repository := range b.Repositories {
		if !repository.Enabled {
			continue
		}

		if err := repository.Dbg(); err != nil {
			panic(err)
		}
	}

	fmt.Println("SUCCESS")
}
