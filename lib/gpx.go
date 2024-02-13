package syncmediatrack

import (
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/karrick/godirwalk"
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
	Ele  float64 `xml:"ele"`
}

var (
	DataGPX    map[string]Gpx
	trackValid int
	trackError int
)

func init() {
	DataGPX = make(map[string]Gpx)
}

func ReadGPX(filename string, valid bool) {
	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		log.Fatal(ColorRed(err))
	}

	if !mtype.Is("application/gpx+xml") && !mtype.Is("text/xml") {
		return
	}

	fmt.Printf("Reading: %v \n", filename)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	gpx := Gpx{}
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&gpx); err != nil {
		fmt.Println(ColorYellow("Warning: GPX file could not be processed, error: ", ColorRed(err)))
		return
	}

	var oldtrkptTime time.Time
	var num int
	var oldlat, oldlon float64
	var stopshow bool

	for _, trkpt := range gpx.Trk.Trkseg.Trkpt {
		if len(trkpt.Time) == 0 {
			continue
		}

		trkptTime := GetTimeFromTrkpt(trkpt)
		if trkptTime.IsZero() {
			continue
		}

		if num > 0 && trkptTime.Before(oldtrkptTime) {
			if !stopshow {
				trackError++
				Warning("Warning: GPX file has time stamps out of order.")
				if valid {
					return
				}
			}
			stopshow = true
		}

		if !oldtrkptTime.IsZero() {
			distance := distancePoints(oldlat, oldlon, trkpt.Lat, trkpt.Lon)
			duration := trkptTime.Sub(oldtrkptTime)
			if duration < 0 {
				duration = -duration
			}

			if distance > 500 && duration.Seconds() < 30 {
				fmt.Printf(ColorRed("Distance: %v lat1: %f lon1: %f, lat2: %f lon2:%f sec %f \n"), distance, oldlat, oldlon, trkpt.Lat, trkpt.Lon, duration.Seconds())

				trackError++
				Warning("Warning: GPX file has a distance between points greater than 500 meters.")
			}
		}

		oldtrkptTime = trkptTime
		oldlat = trkpt.Lat
		oldlon = trkpt.Lon

		num++
	}

	if num > 0 || !valid {
		trackValid++
		if Verbose {
			// Print first and last time stamp
			first := gpx.Trk.Trkseg.Trkpt[0]
			last := gpx.Trk.Trkseg.Trkpt[len(gpx.Trk.Trkseg.Trkpt)-1]
			fmt.Printf("First: %v Last: %v\n", GetTimeFromTrkpt(first), GetTimeFromTrkpt(last))
		}

		DataGPX[filename] = gpx
		return
	}

	trackError++

	fmt.Println(ColorYellow("Warning: GPX file does not have time stamps."))
}

func ReadGPXDir(trackDir string, valid bool) {
	err := godirwalk.Walk(trackDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			ReadGPX(path, valid)

			return nil
		},
		Unsorted: false,
	})
	if err != nil {
		fmt.Println(err)
	}
}

func distancePoints(lat1, lon1, lat2, lon2 float64) float64 {
	// Earth radius in meters
	const earthRadius = 6371000
	radiansLat1 := degrees2radians(lat1)
	radiansLon1 := degrees2radians(lon1)
	radiansLat2 := degrees2radians(lat2)
	radiansLon2 := degrees2radians(lon2)
	deltaLat := radiansLat2 - radiansLat1
	deltaLon := radiansLon2 - radiansLon1
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(radiansLat1)*math.Cos(radiansLat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c
	return distance
}

func degrees2radians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func GetTimeFromTrkpt(trkpt Trkpt) time.Time {
	if len(trkpt.Time) == 0 {
		return time.Time{}
	}

	t, err := time.Parse("2006-01-02T15:04:05Z", trkpt.Time)
	if err != nil {
		return time.Time{}
	}

	return UpdateGPSDateTime(t, trkpt.Lat, trkpt.Lon)
}

func ReadTracks(track string, valid bool) {
	fileInfo, err := os.Stat(track)
	if err != nil {
		log.Fatal(ColorRed("No open GPX path"))
	}

	Pass("Reading tracks...")

	if fileInfo.IsDir() {
		ReadGPXDir(track, valid)
	} else {
		ReadGPX(track, valid)
	}

	if len(DataGPX) == 0 {
		Warning("There is no track processed")
	}

	if trackError == 0 {
		fmt.Printf(ColorGreen("Processed %d track(s)\n"), trackValid)
	} else {
		fmt.Printf(ColorYellow("Processed %d track(s), %d with error(s)\n"), trackValid, trackError)
	}
}
