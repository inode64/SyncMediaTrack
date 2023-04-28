package syncmediatrack

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func FileIsMedia(filename string) bool {
	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		return false
	}

	return strings.Contains(mtype.String(), "video/") || strings.Contains(mtype.String(), "image/")
}

func FileIsVideo(filename string) bool {
	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		return false
	}

	return strings.Contains(mtype.String(), "video/")
}
