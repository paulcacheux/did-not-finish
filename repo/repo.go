package repo

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

	"golang.org/x/crypto/openpgp"
	"gopkg.in/ini.v1"

	"github.com/paulcacheux/did-not-finish/internal/utils"
	"github.com/paulcacheux/did-not-finish/types"
	"github.com/sassoftware/go-rpmutils"
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

func ReadFromDir(repoDir string, varsReplacer *strings.Replacer) ([]Repo, error) {
	repoFiles, err := filepath.Glob(utils.HostEtcJoin(repoDir, "*.repo"))
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

type PkgMatchFunc = func(*types.Package) bool

func (r *Repo) FetchPackage(pkgMatcher PkgMatchFunc) (*types.Package, []byte, error) {
	repoMd, err := r.FetchRepoMD()
	if err != nil {
		return nil, nil, err
	}

	pkgs, err := r.FetchPackagesLists(repoMd)
	if err != nil {
		return nil, nil, err
	}

	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, nil, err
	}

	var entityList openpgp.EntityList
	if r.GpgCheck {
		gpgKeyUrl, err := url.Parse(r.GpgKey)
		if err != nil {
			return nil, nil, err
		}
		if gpgKeyUrl.Scheme != "file" {
			return nil, nil, fmt.Errorf("only file scheme are supported for gpg key: %s", r.GpgKey)
		}

		publicKeyFile, err := os.Open(utils.HostEtcJoin(gpgKeyUrl.Path))
		if err != nil {
			return nil, nil, err
		}

		el, err := openpgp.ReadArmoredKeyRing(publicKeyFile)
		if err != nil {
			return nil, nil, err
		}
		entityList = el
	}

	for _, pkg := range pkgs {
		if pkgMatcher(pkg) {
			pkgUrl, err := utils.UrlJoinPath(fetchURL, pkg.Location.Href)
			if err != nil {
				return nil, nil, err
			}

			resp, err := http.Get(pkgUrl)
			if err != nil {
				return nil, nil, err
			}
			defer resp.Body.Close()

			pkgRpm, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, nil, err
			}

			if r.GpgCheck {
				rpmReader := bytes.NewReader(pkgRpm)
				_, _, err := rpmutils.Verify(rpmReader, entityList)
				if err != nil {
					return nil, nil, err
				}
			}

			return pkg, pkgRpm, nil
		}
	}

	// no error, but no package found either
	return nil, nil, nil
}

func (r *Repo) FetchRepoMD() (*types.Repomd, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, err
	}

	repoMDUrl, err := utils.UrlJoinPath(fetchURL, "repodata/repomd.xml")
	if err != nil {
		return nil, err
	}

	repoMd, err := utils.GetAndUnmarshalXML[types.Repomd](repoMDUrl, nil)
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

		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		mirrors = append(mirrors, sc.Text())
	}

	if len(mirrors) == 0 {
		return "", fmt.Errorf("no mirror available")
	}

	r.BaseURL = mirrors[0]
	return r.BaseURL, nil
}

func (r *Repo) FetchPackagesLists(repoMd *types.Repomd) ([]*types.Package, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, err
	}

	allPackages := make([]*types.Package, 0)

	for _, d := range repoMd.Data {
		if d.Type == "primary" {
			primaryURL, err := utils.UrlJoinPath(fetchURL, d.Location.Href)
			if err != nil {
				return nil, err
			}

			metadata, err := utils.GetAndUnmarshalXML[types.Metadata](primaryURL, &d.OpenChecksum)
			if err != nil {
				return nil, err
			}

			allPackages = append(allPackages, metadata.Packages...)
		}
	}

	return allPackages, nil
}
