package cmd

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/spf13/cobra"
	gogpx "github.com/twpayne/go-gpx"
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

var updateHeader bool

func init() {
	rootCmd.AddCommand(updateTrackCmd)
	rootCmd.PersistentFlags().BoolVar(&updateHeader, "updateheader", false, "Store the old filename in header.name")
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
		// split path and basename from filename
		path := filepath.Dir(filename)
		basename := filepath.Base(filename)

		fmt.Printf("[%v] -> ", basename)

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
		trkptTime = syncmediatrack.UpdateGPSDateTime(trkptTime, trkpt.Lat, trkpt.Lon)

		newfilename := trkptTime.Format("2006_01_02_15_04_mon")

		if geoservice {
			loc, _ := syncmediatrack.ReverseLocation(trkpt)
			if len(loc) != 0 {
				// remove '/' from loc
				loc = syncmediatrack.GeonameCleanup(loc)

				newfilename = fmt.Sprintf("%s_%s", newfilename, loc)
			}
		}

		newfilename = fmt.Sprintf("%s.gpx", newfilename)

		fmt.Println(newfilename)

		if dryRun {
			continue
		}

		if updateHeader {
			updateHeaderName(filename, basename)
		}

		newfilename = fmt.Sprintf("%s/%s", path, newfilename)

		// rename filename to newfilename
		err = os.Rename(filename, newfilename)
		if err != nil {
			fmt.Println(syncmediatrack.ColorRed(err))

			continue
		}

		// update time of the newfilename from trkptTime
		err = os.Chtimes(newfilename, trkptTime, trkptTime)
		if err != nil {
			fmt.Println(syncmediatrack.ColorRed(err))

			continue
		}
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

func updateHeaderName(filename string, basename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	g, err := gogpx.Read(f)
	if err != nil {
		return
	}
	if g.Metadata == nil {
		g.Metadata = &gogpx.MetadataType{}
	}

	if g.Metadata.Name == "" {
		// remove extension from basename
		g.Metadata.Name = basename[:len(basename)-len(filepath.Ext(basename))]

		f, err = os.Create(filename)
		if err != nil {
			fmt.Println(syncmediatrack.ColorRed(err))
			return
		}
		defer f.Close()

		// write xml header
		_, err = f.WriteString(xml.Header)
		if err != nil {
			fmt.Println(syncmediatrack.ColorRed(err))
			return
		}

		if err := g.WriteIndent(f, "", "  "); err != nil {
			fmt.Println(syncmediatrack.ColorRed(err))
		}
	}
}
