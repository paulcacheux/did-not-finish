package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sassoftware/go-rpmutils"
	"golang.org/x/crypto/openpgp"
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

	pkgs, err := r.FetchPackages(repoMd)
	if err != nil {
		return err
	}

	fetchURL, err := r.FetchURL()
	if err != nil {
		return err
	}

	var entityList openpgp.EntityList
	if r.GpgCheck {
		gpgKeyUrl, err := url.Parse(r.GpgKey)
		if err != nil {
			return err
		}
		if gpgKeyUrl.Scheme != "file" {
			return fmt.Errorf("only file scheme are supported for gpg key: %s", r.GpgKey)
		}

		publicKeyFile, err := os.Open(gpgKeyUrl.Path)
		if err != nil {
			return err
		}

		el, err := openpgp.ReadArmoredKeyRing(publicKeyFile)
		if err != nil {
			return err
		}
		entityList = el
	}

	for _, pkg := range pkgs {
		if strings.HasPrefix(pkg.Name, "kernel-headers") {
			pkgUrl, err := url.JoinPath(fetchURL, pkg.Location.Href)
			if err != nil {
				return err
			}

			resp, err := http.Get(pkgUrl)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			pkgRpm, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if r.GpgCheck {
				rpmReader := bytes.NewReader(pkgRpm)
				_, _, err := rpmutils.Verify(rpmReader, entityList)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *Repo) FetchRepoMD() (*Repomd, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, err
	}

	repoMDUrl, err := url.JoinPath(fetchURL, "repodata/repomd.xml")
	if err != nil {
		return nil, err
	}

	repoMd, err := GetAndUnmarshalXML[Repomd](repoMDUrl, nil)
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

func (r *Repo) FetchPackages(repoMd *Repomd) ([]Package, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, err
	}

	allPackages := make([]Package, 0)

	for _, d := range repoMd.Data {
		if d.Type == "primary" {
			primaryURL, err := url.JoinPath(fetchURL, d.Location.Href)
			if err != nil {
				return nil, err
			}

			metadata, err := GetAndUnmarshalXML[Metadata](primaryURL, &d.OpenChecksum)
			if err != nil {
				return nil, err
			}

			allPackages = append(allPackages, metadata.Packages...)
		}
	}

	return allPackages, nil
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
