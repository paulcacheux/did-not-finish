package main

import (
	"bufio"
	"fmt"
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
	repoMd, err := r.FetchRepoMD()
	if err != nil {
		return err
	}

	if err := r.FetchPrimary(repoMd); err != nil {
		return err
	}

	return nil
}

func (r *Repo) FetchRepoMD() (*Repomd, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, err
	}

	repoMDUrl := fetchURL + "repodata/repomd.xml"
	repoMd, err := GetAndUnmarshalXML[Repomd](repoMDUrl)
	if err != nil {
		return nil, err
	}

	return repoMd, nil
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

func (r *Repo) FetchPrimary(repoMd *Repomd) error {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return err
	}

	for _, d := range repoMd.Data {
		if d.Type == "primary" {
			primaryURL := fetchURL + d.Location.Href

			metadata, err := GetAndUnmarshalXML[Metadata](primaryURL)
			if err != nil {
				return err
			}

			for _, pkg := range metadata.Packages {
				if strings.HasPrefix(pkg.Name, "kernel-headers") {
					fmt.Printf("%+v\n", pkg)
				}
			}
		}
	}

	return nil
}

type Repomd struct {
	Data []RepomdData `xml:"data"`
}

type RepomdData struct {
	Type         string   `xml:"type,attr"`
	Size         int      `xml:"size"`
	OpenSize     int      `xml:"open-size"`
	Location     Location `xml:"location"`
	Checksum     Checksum `xml:"checksum"`
	OpenChecksum Checksum `xml:"open-checksum"`
}

type Location struct {
	Href string `xml:"href,attr"`
}
