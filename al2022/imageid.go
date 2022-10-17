package al2022

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/paulcacheux/did-not-finish/internal/utils"
)

var imageFilePattern = regexp.MustCompile(`image_file="al2022-\w+-(2022.0.\d{8}).*"`)

func ExtractReleaseVersionFromImageID() (string, error) {
	imageIDPath := utils.HostEtcJoin("/etc/image-id")
	f, err := os.Open(imageIDPath)
	if err != nil {
		return "", err
	}

	liner := bufio.NewScanner(f)
	for liner.Scan() {
		if submatches := imageFilePattern.FindStringSubmatch(liner.Text()); submatches != nil {
			return submatches[1], nil
		}
	}

	if err := liner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("image_file entry not found in %s", imageIDPath)
}
