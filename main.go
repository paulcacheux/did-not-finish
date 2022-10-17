package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/paulcacheux/did-not-finish/al2022"
	"github.com/paulcacheux/did-not-finish/backend"
	"github.com/paulcacheux/did-not-finish/types"
)

func main() {
	var repoPath, varsPaths string

	flag.StringVar(&repoPath, "repos", "/etc/yum.repos.d/", "path to repos")
	flag.StringVar(&varsPaths, "vars", "/etc/dnf/vars/,/etc/yum/vars/", "paths to variables")

	flag.Parse()

	al2022version, err := al2022.ExtractReleaseVersionFromImageID()
	if err != nil {
		panic(err)
	}

	builtinVars, err := backend.ComputeBuiltinVariables(al2022version)
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

		_, _, err := repository.FetchPackage(func(p *types.Package) bool {
			return p.Name == "kernel-headers"
		})
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("SUCCESS")
}
