package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/paulcacheux/did-not-finish/backend"
	"github.com/paulcacheux/did-not-finish/types"
)

func main() {
	var repoPath, varsPaths, releaseVer string

	flag.StringVar(&repoPath, "repos", "/etc/yum.repos.d/", "path to repos")
	flag.StringVar(&varsPaths, "vars", "/etc/dnf/vars/,/etc/yum/vars/", "paths to variables")
	flag.StringVar(&releaseVer, "release-ver", "", "release version")

	flag.Parse()

	builtinVars, err := backend.ComputeBuiltinVariables(releaseVer)
	if err != nil {
		panic(err)
	}

	b, err := backend.NewBackend(repoPath, strings.Split(varsPaths, ","), builtinVars)
	if err != nil {
		panic(err)
	}

	_, _, err = b.FetchPackage(func(p *types.Package) bool {
		return p.Name == "kernel-headers"
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("SUCCESS")
}
