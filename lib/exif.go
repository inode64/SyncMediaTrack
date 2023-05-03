package syncmediatrack

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/konradit/gopro-utils/telemetry"
	"github.com/konradit/mmt/pkg/videomanipulation"
	"github.com/ringsaturn/tzf"
)

var (
	Verbose bool
	finder  tzf.F
)

func init() {
	var err error
	finder, err = tzf.NewDefaultFinder()
	if err != nil {
		log.Fatal(ColorRed(err))
	}
}

func GetMediaDate(filename string, gps *Trkpt) (time.Time, time.Time, time.Time, error) {
	var atime, etime, gtime time.Time

	f, err := os.Stat(filename)
	if err != nil {
		return atime, etime, gtime, err
	}
	atime = f.ModTime()

	if FileIsVideo(filename) {
		gtime = getTimeFromMP4(filename)
	}

	// create an instance of exiftool
	et, err := exiftool.NewExiftool(exiftool.CoordFormant("%+f"))
	if err != nil {
		return atime, etime, gtime, err
	}
	defer et.Close()

	metas := et.ExtractMetadata(filename)

	gps.Lon, _ = metas[0].GetFloat("GPSLongitude")
	gps.Lat, _ = metas[0].GetFloat("GPSLatitude")
	EleStr, err := metas[0].GetString("GPSAltitude")
	if err == nil {
		re := regexp.MustCompile(`(-?\d+(\.\d{1,4})?) m.*`)
		match := re.FindStringSubmatch(EleStr)

		if len(match) > 1 {
			alt, err := strconv.Atoi(match[1])
			if err == nil {
				gps.Ele = float64(alt)
			}
		}
	}
	if gps.Lon != 0 && gps.Lat != 0 && gtime.IsZero() {
		t, err := metas[0].GetString("GPSDateTime")
		if err == nil {
			gtime, _ = time.Parse("2006:01:02 15:04:05Z", t)
			gtime = UpdateGPSDateTime(gtime, gps.Lat, gps.Lon)
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
			t, err := time.Parse("2006:01:02 15:04:05", val)
			if err == nil {
				return atime, t, gtime, nil
			}
		}
	}

	return atime, etime, gtime, nil
}

func GetClosesGPS(imageTime time.Time, closestPoint *Trkpt) bool {
	var closestDuration time.Duration
	var oldtrkptTime time.Time
	var closestFilename string

	for filename, gpx := range DataGPX {
		closestDuration = 0
		oldtrkptTime = time.Time{}

		first := gpx.Trk.Trkseg.Trkpt[0]
		last := gpx.Trk.Trkseg.Trkpt[len(gpx.Trk.Trkseg.Trkpt)-1]

		if !isBetween(imageTime, GetTimeFromTrkpt(first), GetTimeFromTrkpt(last)) {
			continue
		}

		for _, trkpt := range gpx.Trk.Trkseg.Trkpt {
			trkptTime := GetTimeFromTrkpt(trkpt)
			if trkptTime.IsZero() {
				continue
			}

			duration := imageTime.Sub(trkptTime)
			if duration < 0 {
				duration = -duration
			}

			if closestDuration == 0 || duration < closestDuration {
				*closestPoint = trkpt
				closestDuration = duration
				closestFilename = filename
			}

			if isBetween(imageTime, oldtrkptTime, trkptTime) {
				if Verbose {
					fmt.Printf(" Diff.sec (%.0f [%s]) ", closestDuration.Seconds(), filename)
				}
				return true
			}
			oldtrkptTime = trkptTime
		}
	}

	if oldtrkptTime.IsZero() {
		return false
	}

	if Verbose && closestDuration.Seconds() < 3600 {
		fmt.Printf(" Diff.sec (%.0f [%s]) ", closestDuration.Seconds(), closestFilename)
	}

	if closestDuration.Seconds() > 500 {
		return false
	}

	return true
}

func isBetween(date, start, end time.Time) bool {
	if start.IsZero() || end.IsZero() {
		return false
	}

	return (date.Equal(start) || date.After(start)) && (date.Equal(end) || date.Before(end))
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

func getTimeFromMP4(videoPath string) time.Time {
	vman := videomanipulation.New()
	data, err := vman.ExtractGPMF(videoPath)
	if err != nil {
		return time.Time{}
	}

	reader := bytes.NewReader(*data)

	lastEvent := &telemetry.TELEM{}

	for {
		event, err := telemetry.Read(reader)
		if err != nil && err != io.EOF {
			return time.Time{}
		} else if err == io.EOF || event == nil {
			break
		}

		if lastEvent.IsZero() {
			*lastEvent = *event
			event.Clear()
			continue
		}

		err = lastEvent.FillTimes(event.Time.Time)
		if err != nil {
			return time.Time{}
		}

		telems := lastEvent.ShitJson()
		for _, telem := range telems {
			if telem.Latitude != 0 && telem.Longitude != 0 {
				t := time.UnixMicro(telem.TS)
				return UpdateGPSDateTime(t, telem.Latitude, telem.Longitude)
			}
		}
		*lastEvent = *event
	}

	return time.Time{}
}

func UpdateGPSDateTime(gpsDateTime time.Time, lat float64, lon float64) time.Time {
	if lat == 0 && lon == 0 {
		return gpsDateTime
	}

	zone := finder.GetTimezoneName(lon, lat)

	if zone == "" {
		return gpsDateTime
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		return gpsDateTime
	}

	trkptTime := gpsDateTime.In(loc)

	// remove timezone from time
	trkptTime, _ = time.Parse("2006-01-02 15:04:05", trkptTime.Format("2006-01-02 15:04:05"))

	return trkptTime
}
