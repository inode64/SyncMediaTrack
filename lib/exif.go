package syncmediatrack

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/barasher/go-exiftool"
)

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
				gps.Ele = float64(alt)
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
			if len(trkpt.Time) == 0 {
				continue
			}
			trkptTime, err := time.Parse("2006-01-02T15:04:05Z", trkpt.Time)
			if err != nil {
				fmt.Printf(ColorRed(err) + "\n")
				continue
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
		log.Fatal(ColorRed(err))
	}

	fileInfo.SetString("GPSDateStamp", gpsTime.Format("2006:01:02"))
	fileInfo.SetString("GPSTimeStamp", gpsTime.Format("15:04:05,00"))

	// Update latitude, longitude, and elevation values
	fileInfo.SetFloat("GPSLatitude", gps.Lat)
	fileInfo.SetFloat("GPSLongitude", gps.Lon)
	fileInfo.SetInt("GPSAltitude", int64(gps.Ele))
	fileInfo.SetString("GPSAltitudeRef", "above sea level")

	fileInfo.SetString("GPSLatitudeRef", latRef)
	fileInfo.SetString("GPSLongitudeRef", lonRef)

	// Write the new metadata to the file.
	et.WriteMetadata([]exiftool.FileMetadata{fileInfo})

	return nil
}
