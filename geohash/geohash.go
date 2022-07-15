// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// This file implements the main xkcd Geohashing algorithm.

package geohash

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// GeoHashProvider to calculate Geohashing locations.
type GeoHashProvider struct {
	DjiaProvider DowJonesIndustrialAvgProvider
}

// Geo hash for a given location, latitude and longitude reduced to an integer,
// and a date.
func (provider *GeoHashProvider) Geo(latArea, lonArea int, date time.Time, ctx context.Context) (lat, lon float64, err error) {
	queryDate := date
	if lonArea > -30 {
		queryDate = date.Add(-24 * time.Hour)
	}

	djia, err := provider.DjiaProvider.Get(queryDate, ctx)
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

// Global hash for a given date.
func (provider *GeoHashProvider) Global(date time.Time, ctx context.Context) (lat, lon float64, err error) {
	lat, lon, err = provider.Geo(0, 0, date, ctx)
	if err != nil {
		return
	}

	lat = lat*180.0 - 90.0
	lon = lon*360.0 - 180.0

	return
}
