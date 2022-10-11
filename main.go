package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

const localRepoPath = "./fixtures/repos"
const localVarsPath = "./fixtures/vars"

var baseVariables = map[string]string{
	"arch":       "aarch64",
	"basearch":   "aarch64",
	"releasever": "2022.0.20220928",
}

type Repo struct {
	SectionName string
	Name        string
	BaseURL     string
	MirrorList  string
	Enabled     bool
	GpgCheck    bool
	GpgKey      string
}

func main() {
	vars, err := readVars(localVarsPath)
	if err != nil {
		panic(err)
	}
	varsReplacer := buildVarsReplacer(baseVariables, vars)

	repos, err := readRepositories(localRepoPath, varsReplacer)
	if err != nil {
		panic(err)
	}

	fmt.Println(repos)
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

func readRepositories(repoDir string, varsReplacer *strings.Replacer) ([]Repo, error) {
	repoFiles, err := filepath.Glob(filepath.Join(repoDir, "*.repo"))
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, 0)
	for _, repoFile := range repoFiles {
		cfg, err := ini.Load(repoFile)
		if err != nil {
			return nil, err
		}

		for _, section := range cfg.Sections() {
			if section.Name() == "DEFAULT" {
				continue
			}

			repo := Repo{}
			repo.SectionName = section.Name()
			repo.Name = varsReplacer.Replace(section.Key("name").String())
			repo.BaseURL = varsReplacer.Replace(section.Key("baseurl").String())
			repo.MirrorList = varsReplacer.Replace(section.Key("mirrorlist").String())
			repo.Enabled, _ = section.Key("enabled").Bool()
			repo.GpgCheck, _ = section.Key("gpgcheck").Bool()
			repo.GpgKey = varsReplacer.Replace(section.Key("gpgkey").String())

			repos = append(repos, repo)
		}
	}
	return repos, nil
}
