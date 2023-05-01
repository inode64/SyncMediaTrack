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
		re := regexp.MustCompile(`(\d+(\.\d+)?) m.*`)
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
			gtime = updateGPSDateTime(gtime, gps.Lat, gps.Lon)
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

	for _, gpx := range DataGPX {
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
				*closestPoint = trkpt
				closestDuration = duration
			}
		}

		if closestDuration.Seconds() > 30 {
			return false
		}
	}

	return true
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
				return updateGPSDateTime(t, telem.Latitude, telem.Longitude)
			}
		}
		*lastEvent = *event
	}

	return time.Time{}
}

func updateGPSDateTime(gpsDateTime time.Time, lat float64, lon float64) time.Time {
	if lat == 0 && lon == 0 {
		return gpsDateTime
	}

	finder, err := tzf.NewDefaultFinder()
	if err != nil {
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

	return gpsDateTime.In(loc)
}
