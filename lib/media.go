package syncmediatrack

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func FileIsMedia(filename string) bool {
	allowed := []string{"image/jpeg", "video/mp4"}

	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		return false
	}

	return mimetype.EqualsAny(mtype.String(), allowed...)
}

func FileIsVideo(filename string) bool {
	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		return false
	}

	return strings.Contains(mtype.String(), "video")
}
