package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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

func (r *Repo) Dbg() error {
	if !r.Enabled {
		return nil
	}

	fetchURL, err := r.FetchURL()
	if err != nil {
		return err
	}

	repoMDUrl := fetchURL + "repodata/repomd.xml"
	resp, err := http.Get(repoMDUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var repoMd Repomd
	if err := xml.Unmarshal(content, &repoMd); err != nil {
		return err
	}

	fmt.Printf("%+v\n", repoMd)
	return nil
}

func (r *Repo) FetchURL() (string, error) {
	if r.BaseURL != "" || r.MirrorList == "" {
		return r.BaseURL, nil
	}

	resp, err := http.Get(r.MirrorList)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	mirrors := make([]string, 0)
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if sc.Err() != nil {
			return "", err
		}

		mirrors = append(mirrors, sc.Text())
	}

	if len(mirrors) == 0 {
		return "", fmt.Errorf("no mirror available")
	}

	r.BaseURL = mirrors[0]
	return r.BaseURL, nil
}

type Repomd struct {
	Data []RepomdData `xml:"data"`
}

type RepomdData struct {
	Type     string   `xml:"type,attr"`
	Size     int      `xml:"size"`
	OpenSize int      `xml:"open-size"`
	Location Location `xml:"location"`
	Checksum Checksum `xml:"checksum"`
}

type Location struct {
	Href string `xml:"href,attr"`
}

type Checksum struct {
	Hash string `xml:",chardata"`
	Type string `xml:"type,attr"`
}
