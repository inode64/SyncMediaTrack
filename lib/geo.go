package syncmediatrack

import (
	"fmt"
	"strings"

	"github.com/codingsince1985/geo-golang/openstreetmap"
)

var DefaultCountry string

func ReverseLocation(location Trkpt) (string, error) {
	service := openstreetmap.Geocoder()

	address, err := service.ReverseGeocode(location.Lat, location.Lon)
	if err != nil {
		return "", err
	}

	if len(address.City) < 9 && address.State != "" {
		if DefaultCountry == address.CountryCode {
			return fmt.Sprintf("%s %s", address.City, address.State), nil
		}

		return fmt.Sprintf("%s %s %s", address.City, address.State, address.Country), nil
	}

	if DefaultCountry == address.CountryCode {
		return address.City, nil
	}

	return fmt.Sprintf("%s %s", address.City, address.Country), nil
}

func GeonameCleanup(input string) string {
	repl := strings.NewReplacer("/", "_", ":", "_", "\\", "_", ".", "_")
	return repl.Replace(strings.TrimSpace(input))
}
