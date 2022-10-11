package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func GetAndUnmarshalXML[T any](url string, checksum *Checksum) (*T, error) {
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

	if checksum != nil {
		if err := verifyChecksum(content, checksum); err != nil {
			return nil, err
		}
	}

	var res T
	if err := xml.Unmarshal(content, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func verifyChecksum(content []byte, checksum *Checksum) error {
	if checksum.Type != "sha256" {
		return fmt.Errorf("unsupported sha type: %s", checksum.Type)
	}

	contentSum := sha256.Sum256(content)
	if checksum.Hash != fmt.Sprintf("%x", contentSum) {
		return errors.New("failed checksum")
	}

	return nil
}
