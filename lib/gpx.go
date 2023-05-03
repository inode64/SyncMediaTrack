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
	fmt.Printf("Reading: %v \n", filename)

	mtype, err := mimetype.DetectFile(filename)
	if err != nil {
		log.Fatal(ColorRed(err))
	}

	if !mtype.Is("application/gpx+xml") && !mtype.Is("text/xml") {
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
		fmt.Println(ColorYellow("Warning: GPX file could not be processed, error: ", ColorRed(err)))
		return
	}

	var oldtrkptTime time.Time
	var num int
	var oldlat, oldlon float64

	for _, trkpt := range gpx.Trk.Trkseg.Trkpt {
		if len(trkpt.Time) == 0 {
			continue
		}

		trkptTime, err := time.Parse("2006-01-02T15:04:05Z", trkpt.Time)
		if err != nil {
			continue
		}
		trkptTime = UpdateGPSDateTime(trkptTime, trkpt.Lat, trkpt.Lon)
		if trkptTime.IsZero() {
			continue
		}

		if num > 0 && trkptTime.Before(oldtrkptTime) {
			trackError++
			fmt.Println(ColorYellow("Warning: GPX file has time stamps out of order."))
			return
		}

		if !oldtrkptTime.IsZero() {
			distance := distancePoints(oldlat, oldlon, trkpt.Lat, trkpt.Lon)

			if distance > 21000 || !valid {
				trackError++
				fmt.Println(ColorYellow("Warning: GPX file has a distance between points greater than 1km."))
				return
			}
		}

		oldtrkptTime = trkptTime
		oldlat = trkpt.Lat
		oldlon = trkpt.Lon

		num++
	}

	if num > 0 || !valid {
		trackValid++

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
	const earthRadius = 6371000 // meters
	radiansLat1 := grados2radians(lat1)
	radiansLon1 := grados2radians(lon1)
	radiansLat2 := grados2radians(lat2)
	radiansLon2 := grados2radians(lon2)
	deltaLat := radiansLat2 - radiansLat1
	deltaLon := radiansLon2 - radiansLon1
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(radiansLat1)*math.Cos(radiansLat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c
	return distance
}

func grados2radians(grados float64) float64 {
	return grados * math.Pi / 180
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

	fmt.Println("Reading tracks...")

	if fileInfo.IsDir() {
		ReadGPXDir(track, valid)
	} else {
		ReadGPX(track, valid)
	}

	if len(DataGPX) == 0 {
		log.Fatal(ColorRed("There is no track processed"))
	}

	if trackError != 0 {
		fmt.Printf(ColorYellow("Processed %d track(s), %d with error(s)\n"), trackValid, trackError)
	} else {
		fmt.Printf("Processed %d track(s)\n", trackValid)
	}
}
