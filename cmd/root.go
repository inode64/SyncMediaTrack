package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

var (
	dryRun     bool
	force      bool
	geoservice bool
	mediaDir   string
	track      string
)

var rootCmd = &cobra.Command{
	Use:     "SyncMediaTack",
	Short:   "Synchronize Media Data from track GPX",
	Long:    `Using a gpx track, analyze a directory with images or movies and add the GPS positions`,
	Args:    cobra.MinimumNArgs(1),
	Version: "1.1",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			mediaDir = args[0]
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Performs the actions without writing to the files")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "Force update even overwriting previous GPS data")
	rootCmd.PersistentFlags().BoolVar(&geoservice, "geoservice", false, "Show location from GPS position from geocoding service of openstreetmap")
	rootCmd.PersistentFlags().StringVar(&track, "track", "", "GPX track or a directory of GPX tracks")
}

func Execute() {
	var gpsOld syncmediatrack.Trkpt

	cobra.CheckErr(rootCmd.Execute())

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

	err = godirwalk.Walk(mediaDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			var date time.Time

			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			if !syncmediatrack.FileIsMedia(path) {
				return nil
			}

			relPath, err := filepath.Rel(mediaDir, path)
			if err != nil {
				return err
			}
			fmt.Printf("[%v] - ", relPath)

			err = syncmediatrack.GetMediaDate(path, &date, &gpsOld)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			location, err := syncmediatrack.GetClosesGPS(date)
			if err != nil {
				fmt.Println(syncmediatrack.ColorRed(err))
				return nil
			}

			if gpsOld.Lat == 0 && gpsOld.Lon == 0 {
				fmt.Printf("No location")
			} else {
				fmt.Printf("Lat %v Lon %v Ele %v", gpsOld.Lat, gpsOld.Lon, gpsOld.Ele)
			}

			fmt.Printf(" -> Lat %v Lon %v Ele %v ", location.Lat, location.Lon, location.Ele)

			if geoservice {
				loc, _ := syncmediatrack.ReverseLocation(location)
				if len(loc) != 0 {
					fmt.Printf("(%s)", syncmediatrack.ColorGreen(loc))
				}
			}
			if !force && gpsOld.Lat != 0 && gpsOld.Lon != 0 {
				color.Yellow("(no update)")
				return nil
			}

			fmt.Printf("\n")

			if dryRun {
				return nil
			}

			err = syncmediatrack.WriteGPS(location, path)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			return nil
		},
	})
	if err != nil {
		fmt.Println(err)
	}
}
