package syncmediatrack

import (
	"fmt"

	"github.com/codingsince1985/geo-golang/openstreetmap"
)

func ReverseLocation(location Trkpt) (string, error) {
	service := openstreetmap.Geocoder()

	address, err := service.ReverseGeocode(location.Lat, location.Lon)
	if err != nil {
		return "", err
	}

	if len(address.City) < 9 && address.State != "" {
		return fmt.Sprintf("%s %s %s", address.City, address.State, address.Country), nil
	}

	return fmt.Sprintf("%s %s", address.City, address.Country), nil
}
