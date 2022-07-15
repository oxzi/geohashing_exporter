// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package geohash

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"
)

// testDjiaProvider is an DowJonesIndustrialAvgProvider for known test values.
type testDjiaProvider struct{}

func (_ *testDjiaProvider) Get(date time.Time, _ context.Context) (float64, error) {
	switch date.Format("2006-01-02") {
	// https://xkcd.com/426/
	case "2005-05-26":
		return 10458.68, nil

	// https://geohashing.site/geohashing/Globalhash#Example
	case "2005-05-27":
		return 10537.08, nil

	// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_30W_compliance
	case "2008-05-23":
		fallthrough
	case "2008-05-26":
		return 12620.90, nil
	case "2008-05-27":
		return 12479.63, nil
	case "2008-05-28":
		return 12542.90, nil

	// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_the_scientific_notation_bug
	case "2012-02-24":
		fallthrough
	case "2012-02-26":
		return 12981.20, nil

	default:
		return 0.0, fmt.Errorf("unsupported date %v", date)
	}
}

func TestGeoHashProviderGeo(t *testing.T) {
	tests := []struct {
		date    string
		latArea int
		lonArea int
		lat     float64
		lon     float64
	}{
		// Original comic, https://xkcd.com/426/
		{"2005-05-26", 37, -122, 37.857713, -122.544544},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_30W_compliance
		{"2008-05-27", 68, -30, 68.20968, -30.10144},
		{"2008-05-27", 68, -29, 68.12537, -29.57711},
		{"2008-05-28", 68, -30, 68.68745, -30.21221},
		{"2008-05-28", 68, -29, 68.71044, -29.11273},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_the_scientific_notation_bug
		{"2012-02-26", 68, -30, 68.000047, -30.483719},
		{"2012-02-26", 68, -29, 68.000047, -29.483719},
	}

	provider := GeoHashProvider{DjiaProvider: &testDjiaProvider{}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s/%d,%d", test.date, test.latArea, test.lonArea), func(t *testing.T) {
			date, err := time.Parse("2006-01-02", test.date)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			lat, lon, err := provider.Geo(test.latArea, test.lonArea, date, ctx)
			if err != nil {
				t.Fatal(err)
			}

			latDelta := math.Abs(lat - test.lat)
			lonDelta := math.Abs(lon - test.lon)

			if latDelta > 0.00001 || lonDelta > 0.00001 {
				t.Fatalf("expected %f, %f instead of %f, %f", test.lat, test.lon, lat, lon)
			}
		})
	}
}

func TestGeoHashProviderGlobal(t *testing.T) {
	tests := []struct {
		date string
		lat  float64
		lon  float64
	}{
		// https://geohashing.site/geohashing/Globalhash#Example
		{"2005-05-27", 25.67229, 37.29761},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_30W_compliance
		{"2008-05-27", -67.43391, 27.75993},
		{"2008-05-28", 37.87947, -139.41640},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_the_scientific_notation_bug
		{"2012-02-26", -89.99161, -5.86128},
	}

	provider := GeoHashProvider{DjiaProvider: &testDjiaProvider{}}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, err := time.Parse("2006-01-02", test.date)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			lat, lon, err := provider.Global(date, ctx)
			if err != nil {
				t.Fatal(err)
			}

			latDelta := math.Abs(lat - test.lat)
			lonDelta := math.Abs(lon - test.lon)

			if latDelta > 0.00001 || lonDelta > 0.00001 {
				t.Fatalf("expected %f, %f instead of %f, %f", test.lat, test.lon, lat, lon)
			}
		})
	}
}
