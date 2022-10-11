package main

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func GetAndUnmarshalXML[T any](url string) (*T, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var reader io.Reader = resp.Body
	if strings.HasSuffix(url, ".gz") {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var res T
	if err := xml.Unmarshal(content, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
