package repo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
	MetaLink    string
	Enabled     bool
	GpgCheck    bool
	GpgKey      string
}

func ReadFromDir(repoDir string) ([]Repo, error) {
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
			repo.Name = section.Key("name").String()
			repo.BaseURL = section.Key("baseurl").String()
			repo.MirrorList = section.Key("mirrorlist").String()
			repo.MetaLink = section.Key("metalink").String()
			repo.Enabled, _ = section.Key("enabled").Bool()
			repo.GpgCheck, _ = section.Key("gpgcheck").Bool()
			repo.GpgKey = section.Key("gpgkey").String()

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
		el, err := readGPGKey(r.GpgKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read gpg key: %w", err)
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
	return nil, nil, fmt.Errorf("failed to find valid package from repo %s", r.Name)
}

func readGPGKey(gpgKey string) (openpgp.EntityList, error) {
	gpgKeyUrl, err := url.Parse(gpgKey)
	if err != nil {
		return nil, err
	}

	var publicKeyReader io.Reader
	if gpgKeyUrl.Scheme == "file" {
		publicKeyFile, err := os.Open(utils.HostEtcJoin(gpgKeyUrl.Path))
		if err != nil {
			return nil, err
		}
		defer publicKeyFile.Close()
		publicKeyReader = publicKeyFile
	} else if gpgKeyUrl.Scheme == "http" || gpgKeyUrl.Scheme == "https" {
		resp, err := http.Get(gpgKeyUrl.RequestURI())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad status for `%s` : %s", gpgKey, resp.Status)
		}
		publicKeyReader = resp.Body
	} else {
		return nil, fmt.Errorf("only file and http(s) scheme are supported for gpg key: %s", gpgKey)
	}

	return openpgp.ReadArmoredKeyRing(publicKeyReader)
}

const repomdSubpath = "repodata/repomd.xml"

func (r *Repo) FetchRepoMD() (*types.Repomd, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return nil, err
	}

	repoMDUrl := fetchURL
	if !strings.HasSuffix(repoMDUrl, "repomd.xml") {
		withFile, err := utils.UrlJoinPath(fetchURL, repomdSubpath)
		if err != nil {
			return nil, err
		}
		repoMDUrl = withFile
	}

	repoMd, err := utils.GetAndUnmarshalXML[types.Repomd](repoMDUrl, nil)
	if err != nil {
		return nil, err
	}

	return repoMd, nil
}

func (r *Repo) FetchURL() (string, error) {
	if r.BaseURL != "" {
		return r.BaseURL, nil
	}

	if r.MirrorList != "" {
		baseURL, err := fetchURLFromMirrorList(r.MirrorList)
		if err != nil {
			return "", err
		}
		r.BaseURL = baseURL
		return r.BaseURL, nil
	}

	if r.MetaLink != "" {
		url, err := fetchURLFromMetaLink(r.MetaLink)
		if err != nil {
			return "", err
		}
		r.BaseURL = url
		return r.BaseURL, nil
	}

	return "", fmt.Errorf("unable to get a base URL for this repo `%s`", r.Name)
}

func fetchURLFromMirrorList(mirrorListURL string) (string, error) {
	resp, err := http.Get(mirrorListURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status for `%s` : %s", mirrorListURL, resp.Status)
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

	return mirrors[0], nil
}

func fetchURLFromMetaLink(metaLinkURL string) (string, error) {
	metalink, err := utils.GetAndUnmarshalXML[types.MetaLink](metaLinkURL, nil)
	if err != nil {
		return "", err
	}

	for _, file := range metalink.Files.Files {
		if file.Name == "repomd.xml" {
			urls := make([]types.MetaLinkFileResourceURL, 0, len(file.Resources.Urls))
			for _, resUrl := range file.Resources.Urls {
				if resUrl.Protocol == "http" || resUrl.Protocol == "https" {
					urls = append(urls, resUrl)
				}
			}

			if len(urls) == 0 {
				return "", errors.New("no url for `repomd.xml` resource")
			}

			sort.Slice(urls, func(i, j int) bool {
				return urls[j].Preference < urls[i].Preference
			})

			repomdUrl := strings.TrimSuffix(urls[0].URL, repomdSubpath)
			return repomdUrl, nil
		}
	}

	return "", fmt.Errorf("failed to fetch base URL from meta link: %s", metaLinkURL)
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
