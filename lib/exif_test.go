package syncmediatrack

import (
	"log"
	"testing"
	"time"

	"github.com/ringsaturn/tzf"
)

func init() {
	var err error
	finder, err = tzf.NewDefaultFinder()
	if err != nil {
		log.Fatal(err)
	}
}

func TestUpdateGPSDateTime(t *testing.T) {

	t.Run("TestUpdateGPSDateTime", func(t *testing.T) {
		gTime, _ := time.Parse("2006:01:02 15:04:05Z", "2024:01:28 07:46:21Z")

		gps := Trkpt{
			Lat: 39.9660521,
			Lon: -1.0931599,
		}
		result := UpdateGPSDateTime(gTime, gps.Lat, gps.Lon)

		if !gTime.Equal(result) {
			t.Errorf("Expected %v, got %v", gTime, result)
		}

		if result.IsDST() {
			t.Errorf("Time is in DST")
		}

		gTime, _ = time.Parse("2006:01:02 15:04:05Z", "2024:08:28 07:46:21Z")

		result = UpdateGPSDateTime(gTime, gps.Lat, gps.Lon)

		if !gTime.Equal(result) {
			t.Errorf("Expected %v, got %v", gTime, result)
		}

		if !result.IsDST() {
			t.Errorf("Time not in DST")
		}
	})
}
