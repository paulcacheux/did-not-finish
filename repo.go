package main

import (
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type Repo struct {
	SectionName string
	Name        string
	BaseURL     string
	MirrorList  string
	Enabled     bool
	GpgCheck    bool
	GpgKey      string
}

func ReadRepositories(repoDir string, varsReplacer *strings.Replacer) ([]Repo, error) {
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
