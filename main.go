package main

import (
	"fmt"
	"path/filepath"

	"gopkg.in/ini.v1"
)

const localRepoPath = "./repos"

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
	repos, err := readRepositories(localRepoPath)
	if err != nil {
		panic(err)
	}

	fmt.Println(repos)
}

func readRepositories(repoPath string) ([]Repo, error) {
	repoFiles, err := filepath.Glob(filepath.Join(repoPath, "*.repo"))
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
			repo.Name = section.Key("name").String()
			repo.BaseURL = section.Key("baseurl").String()
			repo.MirrorList = section.Key("mirrorlist").String()
			repo.Enabled, _ = section.Key("enabled").Bool()
			repo.GpgCheck, _ = section.Key("gpgcheck").Bool()
			repo.GpgKey = section.Key("gpgkey").String()

			repos = append(repos, repo)
		}
	}
	return repos, nil
}
