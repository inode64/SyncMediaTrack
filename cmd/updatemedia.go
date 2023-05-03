package cmd

import (
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"time"

	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

type mediaGPS struct {
	Lat  float64
	Lon  float64
	Time time.Time
	Ele  float64
}

var (
	gpsOld      syncmediatrack.Trkpt
	mediaDir    string
	mediaValid  int
	mediaError  int
	mediaUpdate int

	fileGPS   = map[string]mediaGPS{}
	fileNoGPS = map[string]mediaGPS{}
)

var updateMediaCmd = &cobra.Command{
	Use:   "updatemedia",
	Short: "Synchronize Media Data from track GPX",
	Long:  `Using a gpx track, analyze a directory with images or movies and add the GPS positions`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			mediaDir = args[0]
		}
		MExecute()
	},
}

// crear una variable para añadir la posicion gps a los ficheros usando como indice el nombre de fichero

func init() {
	rootCmd.AddCommand(updateMediaCmd)

	fileGPS = make(map[string]mediaGPS)
	fileNoGPS = make(map[string]mediaGPS)
}

func MExecute() {
	// check if ffmpeg is installed
	if !FfmpegInstalled() {
		syncmediatrack.Warning("Ffmpeg is not installed, checking the GPS position and time of Gopro videos will not be performed")
	}

	syncmediatrack.ReadTracks(track, true)

	syncmediatrack.Pass("Reading medias...")
	syncmediatrack.Pass("First pass...")

	err := godirwalk.Walk(mediaDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			var location syncmediatrack.Trkpt

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
				fileNoGPS[path] = mediaGPS{Time: date}

				fmt.Printf("No location ")
			} else {
				fileGPS[path] = mediaGPS{Lat: gpsOld.Lat, Lon: gpsOld.Lon, Ele: gpsOld.Ele, Time: date}

				fmt.Printf("Lat %v Lon %v Ele %v ", gpsOld.Lat, gpsOld.Lon, gpsOld.Ele)
			}

			if !syncmediatrack.GetClosesGPS(date, &location) {
				if gpsOld.Lat != 0 && gpsOld.Lon != 0 {
					fmt.Println()
				} else {
					fmt.Println(syncmediatrack.ColorRed("(There is no close time to obtain the GPS position)"))
				}

				return nil
			}

			fmt.Printf("-> Lat %v Lon %v Ele %v ", location.Lat, location.Lon, location.Ele)

			if geoservice {
				loc, _ := syncmediatrack.ReverseLocation(location)
				if len(loc) != 0 {
					fmt.Printf("(%s)", syncmediatrack.ColorGreen(loc))
				}
			}
			if !force && gpsOld.Lat != 0 && gpsOld.Lon != 0 {
				fmt.Println("")
				return nil
			}

			fmt.Println(syncmediatrack.ColorGreen("(updating)"))

			mediaUpdate++

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
		Unsorted: false,
	})
	if err != nil {
		syncmediatrack.Warning(err.Error())
	}

	syncmediatrack.Pass("Second pass...")

	for filename, media := range fileNoGPS {
		var location mediaGPS
		var tlocation syncmediatrack.Trkpt

		fmt.Printf("[%v] - %s - No location -> ", filename, media.Time.Format("02/01/2006 15:04:05"))

		if !getClosesMedia(media, &location) {
			fmt.Println(syncmediatrack.ColorRed("(There is no close time to obtain the GPS position)"))
			continue
		}

		fmt.Printf("Lat %v Lon %v Ele %v ", media.Lat, media.Lon, media.Ele)

		tlocation.Lat = location.Lat
		tlocation.Lon = location.Lon
		tlocation.Ele = location.Ele
		tlocation.Time = location.Time.Format("2006-01-02T15:04:05Z")

		if geoservice {
			loc, _ := syncmediatrack.ReverseLocation(tlocation)
			if len(loc) != 0 {
				fmt.Printf("(%s)", syncmediatrack.ColorGreen(loc))
			}
		}
		if !force && gpsOld.Lat != 0 && gpsOld.Lon != 0 {
			fmt.Println("")
			continue
		}

		fmt.Println(syncmediatrack.ColorGreen("(updating)"))

		mediaUpdate++

		if dryRun {
			continue
		}

		err = syncmediatrack.WriteGPS(tlocation, filename)
		if err != nil {
			fmt.Println(err)
		}
	}

	if mediaError == 0 {
		fmt.Printf(syncmediatrack.ColorGreen("Processed %d media(s)\n"), mediaValid)
	} else {
		fmt.Printf(syncmediatrack.ColorYellow("Processed %d media(s), %d with error(s)\n"), mediaValid, mediaError)
	}

	if mediaError == 0 {
		fmt.Printf(syncmediatrack.ColorGreen("Updated %d media(s) with GPS position\n"), mediaUpdate)
	} else {
		fmt.Println(syncmediatrack.ColorYellow("No media file has been updated with the GPS position"))
	}
}

func getClosesMedia(media mediaGPS, closestPoint *mediaGPS) bool {
	var closestDuration time.Duration
	var closestFilename string

	for filename, gps := range fileGPS {
		duration := media.Time.Sub(gps.Time)
		if duration < 0 {
			duration = -duration
		}

		if closestDuration == 0 || duration < closestDuration {
			*closestPoint = gps
			closestDuration = duration
			closestFilename = filename
		}
	}

	if syncmediatrack.Verbose && closestDuration.Seconds() < 3600 {
		fmt.Printf(" Diff.sec (%.0f [%s]) ", closestDuration.Seconds(), closestFilename)
	}

	if closestDuration.Seconds() > 180 || len(closestFilename) == 0 {
		return false
	}

	return true
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
