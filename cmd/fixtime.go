package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

type ImageInfo struct {
	Path         string
	atime        time.Time
	etime        time.Time
	gtime        time.Time
	HasGPSDate   bool
	AdjustedDate time.Time
	IsAdjusted   bool
}

var imageFile = map[string]ImageInfo{}

var fixTimeCmd = &cobra.Command{
	Use:   "fixtime",
	Short: "Fix time in image files",
	Long:  `Corrects time in image files using GEO data and other adjacent images to correct time shift`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			mediaDir = args[0]
		}
		fixTimeExecute()
	},
}

func init() {
	rootCmd.AddCommand(fixTimeCmd)

	imageFile = make(map[string]ImageInfo)
}

func fixTimeExecute() {
	syncmediatrack.Pass("Reading medias...")
	syncmediatrack.Pass("First pass...")

	err := godirwalk.Walk(mediaDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			var gpsOld syncmediatrack.Trkpt

			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			if !syncmediatrack.FileIsMedia(path) {
				return nil
			}

			mediaValid++

			relPath, err := filepath.Rel(mediaDir, path)
			if err != nil {
				mediaError++
				return err
			}
			fmt.Printf("[%v] - ", relPath)

			atime, etime, gtime, err := syncmediatrack.GetMediaDate(path, &gpsOld)
			if err != nil {
				mediaError++
				fmt.Println(err)
				return nil
			}

			imageFile[path] = ImageInfo{atime: atime, etime: etime, gtime: gtime}

			mediaUpdate++

			return nil
		},
		Unsorted: false,
	})
	if err != nil {
		syncmediatrack.Warning(err.Error())
	}

	syncmediatrack.Pass("Second pass...")
}
