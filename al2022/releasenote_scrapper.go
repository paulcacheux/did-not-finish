package al2022

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
)

const (
	tocContentURL = "https://docs.aws.amazon.com/linux/al2022/release-notes/toc-contents.json"
)

type TocNode struct {
	Title    string    `json:"title"`
	Href     string    `json:"href"`
	Contents []TocNode `json:"contents"`
}

type releaseNoteExtractor struct {
	versions []string
}

var allPackagesMatcher = regexp.MustCompile(`all-packages-al2022-(\d{8}).html`)

func (e *releaseNoteExtractor) visitNode(node *TocNode) {
	if submatches := allPackagesMatcher.FindStringSubmatch(node.Href); submatches != nil {
		e.versions = append(e.versions, submatches[1])
	}

	for _, sub := range node.Contents {
		e.visitNode(&sub)
	}
}

func ScrapAL2022ReleaseNotes() ([]string, error) {
	resp, err := http.Get(tocContentURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tocData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var toc TocNode
	if err := json.Unmarshal(tocData, &toc); err != nil {
		return nil, err
	}

	extractor := releaseNoteExtractor{}
	extractor.visitNode(&toc)
	result := extractor.versions
	sort.Strings(result)

	return result, nil
}
