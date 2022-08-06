// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// This file implements the main xkcd Geohashing algorithm.

// Package geohash provides a 30W compatible implementation of the XKCD
// Geohashing algorithm.
package geohash

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// ErrW30NotYetAvailable is returned if coordinates should be calculated west of
// 30 deg west before the New York Stock Exchange (NYSE) has opened, 09:30.
//
// https://geohashing.site/geohashing/30W_Time_Zone_Rule
var ErrW30NotYetAvailable = errors.New("coordinates west of 30 deg west are not yet available, 30W rule")

// GeoHashProvider to calculate Geohashing locations.
//
// To get an instance, call GetGeoHashProvider.
type GeoHashProvider struct {
	djiaProvider dowJonesIndustrialAvgProvider
}

// geoHashProviderInstance is the singleton instance of the GeoHashProvider.
var geoHashProviderInstance *GeoHashProvider

// geoHashProviderInstanceLock ensures no data race happens when concurrently
// accessing/creating geoHashProviderInstance.
var geoHashProviderInstanceLock sync.Mutex

// GetGeoHashProvider returns a singleton instance of the GeoHashProvider.
func GetGeoHashProvider() *GeoHashProvider {
	geoHashProviderInstanceLock.Lock()
	defer geoHashProviderInstanceLock.Unlock()

	if geoHashProviderInstance == nil {
		geoHashProviderInstance = &GeoHashProvider{
			djiaProvider: newDjiaCache(),
		}
	}

	return geoHashProviderInstance
}

// normalizeDate based on the geographical location and the NYSE holidays.
//
// If the given date is a normal NYSE working day western of 30W,
// ErrW30NotYetAvailable will be returned.
func (provider *GeoHashProvider) normalizeDate(latArea, lonArea int, date time.Time) (queryDate time.Time, err error) {
	queryDate = date

	if lonArea > -30 {
		queryDate = date.Add(-24 * time.Hour)
	} else if dowHourCheckMarketClosed(date) && !isDowHoliday(date) {
		err = ErrW30NotYetAvailable
		return
	}

	queryDate, err = correctDowDate(queryDate)
	return
}

// Geo hash for a given location, latitude and longitude reduced to an integer,
// and a date.
func (provider *GeoHashProvider) Geo(latArea, lonArea int, date time.Time, ctx context.Context) (lat, lon float64, err error) {
	queryDate, err := provider.normalizeDate(latArea, lonArea, date)
	if err != nil {
		return
	}

	djia, err := provider.djiaProvider.Get(queryDate, ctx)
	if err != nil {
		return
	}

	h := md5.Sum([]byte(fmt.Sprintf("%s-%.2f", date.Format("2006-01-02"), djia)))

	fields := []struct {
		area float64
		hash []byte
		out  *float64
	}{
		{float64(latArea), h[0 : md5.Size/2], &lat},
		{float64(lonArea), h[md5.Size/2 : md5.Size], &lon},
	}

	for _, field := range fields {
		decPlace := float64(binary.BigEndian.Uint64(field.hash)) / math.Pow(2.0, 64.0)
		absPos := math.Abs(field.area) + decPlace
		*field.out = math.Copysign(absPos, field.area)
	}

	return
}

// globalNormalizeDate for Globalhash calculation.
func (provider *GeoHashProvider) globalNormalizeDate(date time.Time) time.Time {
	year, month, day := date.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// Global hash for a given date.
//
// Location information will be stripped to normalize the time.
func (provider *GeoHashProvider) Global(date time.Time, ctx context.Context) (lat, lon float64, err error) {
	normalizedDate := provider.globalNormalizeDate(date)
	lat, lon, err = provider.Geo(0, 0, normalizedDate, ctx)
	if err != nil {
		return
	}

	lat = lat*180.0 - 90.0
	lon = lon*360.0 - 180.0

	return
}

// GeoNext calculates all possible future Geohashes after the given date.
//
// It returns an array of a two dimensional float64 array, representing lat and
// lon. The index of the outer array is offset of days to the requested date
// parameter, e.g., 0 is the requested date, 1 is the following one, and so on.
//
// On weekends or NYSE holidays, the last known Dow Jones Industrial Average
// indicator will be used. For example, on Saturdays western of 30W, both the
// date for tomorrow's Sunday as well as the DJIA value is known. Thus, the
// Geohash's location for the following day can already be calculated.
func (provider *GeoHashProvider) GeoNext(latArea, lonArea int, date time.Time, ctx context.Context) (locs [][]float64, err error) {
	for {
		lat, lon, geoErr := provider.Geo(latArea, lonArea, date, ctx)
		if geoErr != nil {
			return nil, geoErr
		}

		locs = append(locs, []float64{lat, lon})

		baseDate, dateErr := provider.normalizeDate(latArea, lonArea, date)
		if dateErr != nil {
			return nil, dateErr
		}

		date = date.Add(24 * time.Hour)

		compDate, dateErr := provider.normalizeDate(latArea, lonArea, date)
		if errors.Is(dateErr, ErrW30NotYetAvailable) {
			// There is at least one coordinate pair in locs and the next possible
			// day will be a new working day west of 30W, we can stop here.
			break
		} else if dateErr != nil {
			return nil, dateErr
		} else if compDate.After(baseDate) {
			break
		}
	}

	return
}

// GlobalNext calculates all possible future Globalhashes after the given date.
//
// For more information look at the documentation for GeoHashProvider.GeoNext.
func (provider *GeoHashProvider) GlobalNext(date time.Time, ctx context.Context) (locs [][]float64, err error) {
	normalizedDate := provider.globalNormalizeDate(date)
	locs, err = provider.GeoNext(0, 0, normalizedDate, ctx)
	if err != nil {
		return
	}

	for date, latLon := range locs {
		locs[date] = []float64{
			latLon[0]*180.0 - 90.0,
			latLon[1]*360.0 - 180.0,
		}
	}

	return
}
