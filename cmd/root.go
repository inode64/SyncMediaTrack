package cmd

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
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

	// check if ffmpeg is installed
	if !FfmpegInstalled() {
		fmt.Println(syncmediatrack.ColorRed("Ffmpeg is not installed, checking the GPS position and time of Gopro videos will not be performed"))
	}

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
			var location syncmediatrack.Trkpt

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

			atime, etime, gtime, err := syncmediatrack.GetMediaDate(path, &gpsOld)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			if etime.IsZero() {
				compareDates2(atime, gtime, "A")
			} else {
				fmt.Printf("[A] ")
				compareDates(atime, etime, 30)
				compareDates2(etime, gtime, "E")
			}

			date := bestDate(atime, etime, gtime)

			fmt.Printf("| ")

			if gpsOld.Lat == 0 && gpsOld.Lon == 0 {
				fmt.Printf("No location")
			} else {
				fmt.Printf("Lat %v Lon %v Ele %v", gpsOld.Lat, gpsOld.Lon, gpsOld.Ele)
			}
			fmt.Printf(" -> ")

			if !syncmediatrack.GetClosesGPS(date, &location) {
				if gpsOld.Lat != 0 && gpsOld.Lon != 0 {
					fmt.Println(syncmediatrack.ColorYellow("Update not necessary"))
				} else {
					fmt.Println(syncmediatrack.ColorRed("There is no close time to obtain the GPS position"))
				}

				return nil
			}

			fmt.Printf("Lat %v Lon %v Ele %v ", location.Lat, location.Lon, location.Ele)

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

func bestDate(atime time.Time, etime time.Time, gtime time.Time) time.Time {
	if !gtime.IsZero() {
		return gtime
	}
	if !etime.IsZero() {
		return etime
	}

	return atime
}

func compareDates(t1 time.Time, t2 time.Time, sec float64) {
	// remove timezone from time
	t1, _ = time.Parse("2006-01-02 15:04:05", t1.Format("2006-01-02 15:04:05"))
	t2, _ = time.Parse("2006-01-02 15:04:05", t2.Format("2006-01-02 15:04:05"))

	diff := math.Abs(t1.Sub(t2).Seconds())

	if diff > sec {
		fmt.Printf("%s -> ", syncmediatrack.ColorYellow(t1.Format("02/01/2006 15:04:05")))
	} else {
		fmt.Printf("%s -> ", t1.Format("02/01/2006 15:04:05"))
	}
}

func compareDates2(old time.Time, gtime time.Time, prefix string) {
	fmt.Printf("[%s] ", prefix)
	if gtime.IsZero() {
		fmt.Printf("%s ", old.Format("02/01/2006 15:04:05"))
	} else {
		compareDates(old, gtime, 80)
		fmt.Printf("[G] %s ", gtime.Format("02/01/2006 15:04:05"))
	}
}

// Function to check if ffmpeg is installed
func FfmpegInstalled() bool {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return false
	}

	cmd := exec.Command(path, "-version")
	err = cmd.Run()

	return err == nil
}
