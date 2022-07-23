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

// testdjiaProvider is an DowJonesIndustrialAvgProvider for known test values.
type testdjiaProvider struct{}

func (_ *testdjiaProvider) Get(date time.Time, _ context.Context) (float64, error) {
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

	// Test values for NYSE opening hours
	case "2022-07-14":
		return 30451.80, nil
	case "2022-07-15":
		return 30775.37, nil

	default:
		return 0.0, fmt.Errorf("unsupported date %v", date)
	}
}

func TestGeoHashProviderGeo(t *testing.T) {
	locNy := nyseTz()
	locBerlin, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		date    string
		loc     *time.Location
		latArea int
		lonArea int
		isErr   bool
		lat     float64
		lon     float64
	}{
		// Original comic, https://xkcd.com/426/
		{"2005-05-26 09:30", locNy, 37, -122, false, 37.857713, -122.544544},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_30W_compliance
		{"2008-05-27 09:30", locNy, 68, -30, false, 68.20968, -30.10144},
		{"2008-05-27 09:30", locNy, 68, -29, false, 68.12537, -29.57711},
		{"2008-05-28 09:30", locNy, 68, -30, false, 68.68745, -30.21221},
		{"2008-05-28 09:30", locNy, 68, -29, false, 68.71044, -29.11273},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_the_scientific_notation_bug
		{"2012-02-26 09:30", locNy, 68, -30, false, 68.000047, -30.483719},
		{"2012-02-26 09:30", locNy, 68, -29, false, 68.000047, -29.483719},

		// NYSE opening hours in regard of the 30W rule.
		{"2022-07-15 09:30", locNy, 40, -74, false, 40.117527, -74.382255},
		{"2022-07-15 09:00", locNy, 40, -74, true, 0.0, 0.0},
		{"2022-07-15 18:00", locBerlin, 52, 13, false, 52.99140, 13.02058},
		{"2022-07-15 00:00", locBerlin, 52, 13, false, 52.99140, 13.02058},
	}

	provider := GeoHashProvider{djiaProvider: &testdjiaProvider{}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s/%d,%d", test.date, test.latArea, test.lonArea), func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02 15:04", test.date, test.loc)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			lat, lon, err := provider.Geo(test.latArea, test.lonArea, date, ctx)
			if (err != nil) != test.isErr {
				t.Fatalf("expected isErr = %t, err = %v", test.isErr, err)
			} else if test.isErr {
				return
			} else if err != nil {
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
	locNy := nyseTz()
	locBerlin, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		date string
		loc  *time.Location
		lat  float64
		lon  float64
	}{
		// https://geohashing.site/geohashing/Globalhash#Example
		{"2005-05-27 09:30", locNy, 25.67229, 37.29761},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_30W_compliance
		{"2008-05-27 09:30", locNy, -67.43391, 27.75993},
		{"2008-05-28 09:30", locNy, 37.87947, -139.41640},

		// https://geohashing.site/geohashing/30W_Time_Zone_Rule#Testing_for_the_scientific_notation_bug
		{"2012-02-26 09:30", locNy, -89.99161, -5.86128},

		// Global hash should be robust against time zones.
		{"2022-07-16 09:30", locNy, 88.520950, -105.946114},
		{"2022-07-16 09:30", locBerlin, 88.520950, -105.946114},
	}

	provider := GeoHashProvider{djiaProvider: &testdjiaProvider{}}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02 15:04", test.date, test.loc)
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

func TestGeoHashProviderGeoNext(t *testing.T) {
	locNy := nyseTz()
	locBerlin, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		date    string
		loc     *time.Location
		latArea int
		lonArea int
		locs    []float64
	}{
		// No pre-calculation possible, it's a work day.
		{"2022-07-15 00:00", locBerlin, 52, 13, []float64{52.99140, 13.02058}},

		// Pre-calculated weekend in Berlin.
		{"2022-07-16 00:00", locBerlin, 52, 13, []float64{52.99178, 13.20571, 52.11295, 13.07143, 52.87523, 13.85938}},

		// Pre-calculated weekend in west of 30W.
		{"2022-07-16 09:30", locNy, 40, -74, []float64{40.99178, -74.20571, 40.11295, -74.07143}},
	}

	provider := GeoHashProvider{djiaProvider: &testdjiaProvider{}}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02 15:04", test.date, test.loc)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			locs, err := provider.GeoNext(test.latArea, test.lonArea, date, ctx)
			if err != nil {
				t.Fatal(err)
			} else if len(locs) != len(test.locs)/2 {
				t.Fatalf("expected %d locations instead of %d", len(test.locs)/2, len(locs))
			}

			for i, latLon := range locs {
				expLat, expLon := test.locs[2*i], test.locs[2*i+1]

				latDelta := math.Abs(expLat - latLon[0])
				lonDelta := math.Abs(expLon - latLon[1])

				if latDelta > 0.00001 || lonDelta > 0.00001 {
					t.Fatalf("offset %d: expected %f, %f instead of %f, %f", i, expLat, expLon, latLon[0], latLon[1])
				}
			}
		})
	}
}

func TestGeoHashProviderGlobalNext(t *testing.T) {
	locNy := nyseTz()
	locBerlin, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		date string
		loc  *time.Location
		locs []float64
	}{
		// No pre-calculation possible, it's a work day.
		{"2022-07-15 00:00", locBerlin, []float64{88.452771, -172.592008}},

		// Pre-calculated weekend; should be same in each time zone.
		{"2022-07-16 09:30", locNy, []float64{88.520950, -105.946114, -69.669076, -154.283436, 67.541519, 129.376863}},
		{"2022-07-16 09:30", locBerlin, []float64{88.520950, -105.946114, -69.669076, -154.283436, 67.541519, 129.376863}},
	}

	provider := GeoHashProvider{djiaProvider: &testdjiaProvider{}}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02 15:04", test.date, test.loc)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			locs, err := provider.GlobalNext(date, ctx)
			if err != nil {
				t.Fatal(err)
			} else if len(locs) != len(test.locs)/2 {
				t.Fatalf("expected %d locations instead of %d", len(test.locs)/2, len(locs))
			}

			for i, latLon := range locs {
				expLat, expLon := test.locs[2*i], test.locs[2*i+1]

				latDelta := math.Abs(expLat - latLon[0])
				lonDelta := math.Abs(expLon - latLon[1])

				if latDelta > 0.00001 || lonDelta > 0.00001 {
					t.Fatalf("offset %d: expected %f, %f instead of %f, %f", i, expLat, expLon, latLon[0], latLon[1])
				}
			}
		})
	}
}
