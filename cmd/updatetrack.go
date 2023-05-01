package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/spf13/cobra"
)

var updateTrackCmd = &cobra.Command{
	Use:   "updatetrack",
	Short: "Update GPX files",
	Long:  `Updates the name and date of the GPX files using the initial position of the track`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			mediaDir = args[0]
		}
		updateTrackExecute()
	},
}

func init() {
	rootCmd.AddCommand(updateTrackCmd)
}

func updateTrackExecute() {
	fileInfo, err := os.Stat(track)
	if err != nil {
		log.Fatal(syncmediatrack.ColorRed("No open GPX path"))
	}

	if fileInfo.IsDir() {
		syncmediatrack.ReadGPXDir(track)
	} else {
		syncmediatrack.ReadGPX(track)
	}

	if len(syncmediatrack.DataGPX) == 0 {
		log.Fatal(syncmediatrack.ColorRed("There is no track processed"))
	}

	for filename, gpx := range syncmediatrack.DataGPX {
		// quitar la ruta del archivo
		filename := filepath.Base(filename)

		fmt.Printf("[%v] - ", filename)

		trkpt := GetPosFromGPX(gpx)
		if len(trkpt.Time) == 0 {
			fmt.Println(syncmediatrack.ColorRed(err))

			continue
		}

		trkptTime, err := time.Parse("2006-01-02T15:04:05Z", trkpt.Time)
		if err != nil {
			fmt.Println(syncmediatrack.ColorRed(err))

			continue
		}

		fmt.Printf("%s", trkptTime.Format("2006_01_02_15_04_mon"))

		if geoservice {
			loc, _ := syncmediatrack.ReverseLocation(trkpt)
			if len(loc) != 0 {
				fmt.Printf("_%s", syncmediatrack.ColorGreen(loc))
			}
		}
		fmt.Println(".gpx")
	}
}

func GetPosFromGPX(gpx syncmediatrack.Gpx) syncmediatrack.Trkpt {
	for _, trkpt := range gpx.Trk.Trkseg.Trkpt {
		if len(trkpt.Time) == 0 {
			continue
		}

		return trkpt
	}

	return syncmediatrack.Trkpt{}
}
