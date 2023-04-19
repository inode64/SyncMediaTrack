package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/fatih/color"
	"github.com/gabriel-vasile/mimetype"
	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

type Gpx struct {
	XMLName xml.Name `xml:"gpx"`
	Trk     Trk      `xml:"trk"`
}

type Trk struct {
	Trkseg Trkseg `xml:"trkseg"`
}

type Trkseg struct {
	Trkpt []Trkpt `xml:"trkpt"`
}

type Trkpt struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Time string  `xml:"time"`
	Ele  int64   `xml:"ele"`
}

var (
	dryRun   bool
	mediaDir string
	track    string
	dataGPX  []Gpx
	force    bool
)

var rootCmd = &cobra.Command{
	Use:   "SyncMediaTack",
	Short: "Synchronize Media Data from track GPX",
	Long:  `Using a gpx track, analyze a directory with images or movies and add the GPS positions`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			mediaDir = args[0]
		}
	},
}

var colorRed = color.New(color.FgRed).SprintFunc()

func init() {
	rootCmd.PersistentFlags().StringVar(&track, "track", "", "GPX track or a directory of GPX tracks")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "Force update even overwriting previous GPS data")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Performs the actions without writing to the files")
}

func readGPX(filename string) {
	fmt.Printf("Processing: %v \n", filename)

	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	if !mtype.Is("application/gpx+xml") {
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	gpx := Gpx{}
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&gpx); err != nil {
		color.Yellow("Warning: GPX file could not be processed")
		return
	}

	dataGPX = append(dataGPX, gpx)
}

func readGPXDir(trackDir string) {
	err := godirwalk.Walk(trackDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			readGPX(path)

			return nil
		},
		Unsorted: true,
	})
	if err != nil {
		fmt.Println(err)
	}
}

func GetMediaDate(filename string, t *time.Time, gps *Trkpt) error {
	f, err := os.Stat(filename)
	if err != nil {
		return err
	}
	*t = f.ModTime()

	// create an instance of exiftool
	et, err := exiftool.NewExiftool(exiftool.CoordFormant("%+f"))
	if err != nil {
		return err
	}
	defer et.Close()

	metas := et.ExtractMetadata(filename)

	gps.Lon, _ = metas[0].GetFloat("GPSLongitude")
	gps.Lat, _ = metas[0].GetFloat("GPSLatitude")
	EleStr, err := metas[0].GetString("GPSAltitude")
	if err == nil {
		re := regexp.MustCompile(`(\d+) m.*`)
		match := re.FindStringSubmatch(EleStr)

		if len(match) > 1 {
			alt, err := strconv.Atoi(match[1])
			if err == nil {
				gps.Ele = int64(alt)
			}
		}
	}

	// define the list of possible tags to extract date from
	dateTags := []string{"DateTimeOriginal", "DateTime", "DateTimeDigitized"}

	// loop through the tags until a valid date is found
	for _, tag := range dateTags {
		val, err := metas[0].GetString(tag)
		if err != nil {
			continue
		}
		if val != "" {
			date, err := time.Parse("2006:01:02 15:04:05", val)
			if err != nil {
				continue
			}
			*t = date
			return nil
		}
	}

	return nil
}

func GetClosesGPS(imageTime time.Time) (Trkpt, error) {
	var closestPoint Trkpt
	var closestDuration time.Duration

	for _, gpx := range dataGPX {
		for _, trkpt := range gpx.Trk.Trkseg.Trkpt {
			trkptTime, err := time.Parse("2006-01-02T15:04:05Z", trkpt.Time)
			if err != nil {
				log.Fatal(err)
			}

			duration := imageTime.Sub(trkptTime.UTC())
			if duration < 0 {
				duration = -duration
			}

			if closestDuration == 0 || duration < closestDuration {
				closestPoint = trkpt
				closestDuration = duration
			}
		}

		if closestDuration.Seconds() > 30 {
			return closestPoint, errors.New("There is no close time to obtain the GPS position")
		}
	}

	return closestPoint, nil
}

func WriteGPS(gps Trkpt, filename string) error {
	et, err := exiftool.NewExiftool()
	if err != nil {
		return err
	}
	defer et.Close()

	// Extract file metadata
	fileInfos := et.ExtractMetadata(filename)
	if len(fileInfos) == 0 {
		return fmt.Errorf("no metadata found %s", filename)
	}
	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		return fileInfo.Err
	}

	latRef := "N"
	if gps.Lat >= 0 {
		latRef = "S"
	}
	lonRef := "E"
	if gps.Lon >= 0 {
		lonRef = "W"
	}

	gpsTime, err := time.Parse("2006-01-02T15:04:05Z", gps.Time)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo.SetString("GPSDateStamp", gpsTime.Format("2006:01:02"))
	fileInfo.SetString("GPSTimeStamp", gpsTime.Format("15:04:05,00"))

	// Update latitude, longitude, and elevation values
	fileInfo.SetFloat("GPSLatitude", gps.Lat)
	fileInfo.SetFloat("GPSLongitude", gps.Lon)
	fileInfo.SetInt("GPSAltitude", gps.Ele)
	fileInfo.SetString("GPSAltitudeRef", "above sea level")

	fileInfo.SetString("GPSLatitudeRef", latRef)
	fileInfo.SetString("GPSLongitudeRef", lonRef)

	// Write the new metadata to the file.
	et.WriteMetadata([]exiftool.FileMetadata{fileInfo})

	return nil
}

func fileIsMedia(filename string) bool {
	allowed := []string{"image/jpeg", "video/mp4"}

	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		return false
	}

	return mimetype.EqualsAny(mtype.String(), allowed...)
}

func main() {
	var gpsOld Trkpt

	cobra.CheckErr(rootCmd.Execute())

	fileInfo, err := os.Stat(track)
	if err != nil {
		log.Fatal("No open GPX path")
	}

	if fileInfo.IsDir() {
		readGPXDir(track)
	} else {
		readGPX(track)
	}

	if len(dataGPX) == 0 {
		log.Fatal("There is no track processed")
	}

	err = godirwalk.Walk(mediaDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			var date time.Time

			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			if !fileIsMedia(path) {
				return nil
			}

			relPath, err := filepath.Rel(mediaDir, path)
			if err != nil {
				return err
			}
			fmt.Printf("[%v] - ", relPath)

			err = GetMediaDate(path, &date, &gpsOld)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			location, err := GetClosesGPS(date)
			if err != nil {
				fmt.Println(colorRed(err))
				return nil
			}

			if gpsOld.Lat == 0 && gpsOld.Lon == 0 {
				fmt.Printf("No location")
			} else {
				fmt.Printf("Lat %v Lon %v Ele %v", gpsOld.Lat, gpsOld.Lon, gpsOld.Ele)
			}

			fmt.Printf(" -> Lat %v Lon %v Ele %v ", location.Lat, location.Lon, location.Ele)
			if !force && gpsOld.Lat != 0 && gpsOld.Lon != 0 {
				fmt.Printf(" * no update\n")
				return nil
			}

			fmt.Printf("\n")

			if dryRun {
				return nil
			}

			err = WriteGPS(location, path)
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
