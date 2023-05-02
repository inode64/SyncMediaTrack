package syncmediatrack

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"

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

var DataGPX map[string]Gpx

func init() {
	DataGPX = make(map[string]Gpx)
}

func ReadGPX(filename string) {
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

	for _, trkpt := range gpx.Trk.Trkseg.Trkpt {
		if len(trkpt.Time) != 0 {
			DataGPX[filename] = gpx
			return
		}
	}

	fmt.Println(ColorYellow("Warning: GPX file does not have time stamps."))
}

func ReadGPXDir(trackDir string) {
	err := godirwalk.Walk(trackDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			ReadGPX(path)

			return nil
		},
		Unsorted: false,
	})
	if err != nil {
		fmt.Println(err)
	}
}
