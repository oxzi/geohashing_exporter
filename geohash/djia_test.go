// SPDX-FileCopyrightText: 2022, 2023 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package geohash

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestDjiaFetch(t *testing.T) {
	tests := []struct {
		date    string
		success bool
		djia    float64
	}{
		// Valid values with pre-fetched data from the past.
		{"2000-03-14", true, 9957.67},
		{"2011-12-13", true, 12018.66},
		{"2022-01-01", true, 36385.85},

		// Unknown future value, needs to be adjusted in round about thousand years.
		{"3000-01-01", false, 0.0},
		// Date from times before the DJIA.
		{"1885-02-15", false, 0.0},
	}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, err := time.Parse("2006-01-02", test.date)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			djia, err := djiaFetch(date, ctx)
			if (err == nil) != test.success {
				t.Fatalf("unexpected result: %q", err)
			}
			if !test.success {
				return
			}

			delta := math.Abs(djia - test.djia)
			if delta > 0.01 {
				t.Fatalf("expected %f instead of %f", test.djia, djia)
			}
		})
	}
}

func TestDowJonesIndustrialAvgCache(t *testing.T) {
	djiaCache := newDjiaCache()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if l := djiaCache.cache.Len(); l != 0 {
		t.Fatalf("initial cache has size %d", l)
	}

	dateStr := "2022-01-01"
	date, _ := time.Parse("2006-01-02", dateStr)

	if v, ok := djiaCache.cache.Get(dateStr); ok {
		t.Fatalf("entry exists before fetching, %v", v)
	}

	djia, err := djiaCache.Get(date, ctx)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := djiaCache.cache.Get(dateStr); !ok {
		t.Fatal("entry is not in cache")
	} else if djiaCache := v; djiaCache != djia {
		t.Fatalf("%f != %f", djia, djiaCache)
	}

	// Measure access time for cached entry. 1000 HTTP requests should take longer
	// than a second. However, 1000 cache hits shouldn't take any time at all.
	// inb4 this test will be executed on some ancient CPU..
	starTime := time.Now()
	for i := 0; i < 1000; i++ {
		if time.Since(starTime) > time.Second {
			t.Fatal("querying took longer than a second")
		}

		djiaCache, err := djiaCache.Get(date, ctx)
		if err != nil {
			t.Fatal(err)
		} else if djiaCache != djia {
			t.Fatalf("%f != %f", djia, djiaCache)
		}
	}
}
