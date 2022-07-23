// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/oxzi/geohashing_exporter/geohash"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsHandlerParseParams fetches the required GET parameters lat, lon, and
// tz for the metricsHandler HTTP handler.
func metricsHandlerParseParams(r *http.Request) (lat, lon int, tz string, err error) {
	latLonParams := []struct {
		key   string
		field *int
	}{
		{"lat", &lat},
		{"lon", &lon},
	}
	for _, param := range latLonParams {
		*param.field, err = strconv.Atoi(r.URL.Query().Get(param.key))
		if err != nil {
			err = fmt.Errorf("cannot parse `%s` GET parameter as an integer: %v", param.key, err)
			return
		}
	}

	tz = r.URL.Query().Get("tz")
	if tz == "" {
		err = fmt.Errorf("`tz` GET parameter is missing")
		return
	}

	return
}

// metricsHandlerGauges creates and populates the labeled Prometheus gauges for
// the latitude and longitude to be returned in the metricsHandler HTTP handler.
func metricsHandlerGauges(lat, lon int, tz string, ctx context.Context) (latGauge, lonGauge *prometheus.GaugeVec, err error) {
	labels := []string{
		// location describes which geohash is meant, as both the neighboring
		// coordinates and the globalhash is also queried. One of:
		// "nw", "n", "ne", "w", "center", "e", "sw", "s", "se", "global"
		"location",
		// day_offset says how many days the geohash lays in the future.
		"day_offset",
	}

	latGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "geohashing_lat",
			Help: "Latitude of the geohash.",
		},
		labels,
	)
	lonGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "geohashing_lon",
			Help: "Longitude of the geohash.",
		},
		labels,
	)

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return
	}
	localTime := time.Now().In(loc)

	geoLocs := []struct {
		name string
		lat  int
		lon  int
	}{
		{"nw", lat + 1, lon - 1},
		{"n", lat + 1, lon},
		{"ne", lat + 1, lon + 1},
		{"w", lat, lon - 1},
		{"center", lat, lon},
		{"e", lat, lon + 1},
		{"sw", lat - 1, lon - 1},
		{"s", lat - 1, lon},
		{"se", lat - 1, lon + 1},
	}
	for _, geoLoc := range geoLocs {
		locs, locsErr := geohash.GetGeoHashProvider().GeoNext(geoLoc.lat, geoLoc.lon, localTime, ctx)
		if locsErr != nil {
			err = locsErr
			return
		}

		for i, loc := range locs {
			label := prometheus.Labels{"location": geoLoc.name, "day_offset": fmt.Sprintf("%d", i)}
			latGauge.With(label).Set(loc[0])
			lonGauge.With(label).Set(loc[1])
		}
	}

	globalLocs, err := geohash.GetGeoHashProvider().GlobalNext(localTime, ctx)
	if err != nil {
		return
	}
	for i, loc := range globalLocs {
		label := prometheus.Labels{"location": "global", "day_offset": fmt.Sprintf("%d", i)}
		latGauge.With(label).Set(loc[0])
		lonGauge.With(label).Set(loc[1])
	}

	return
}

// metricsHandler is a HTTP handler function for a Prometheus exporter, listing
// the next geohashes coordinates in the requested coordinate window, the
// neighboring ones and for the globalhash.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	lat, lon, tz, err := metricsHandlerParseParams(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	latGauge, lonGauge, err := metricsHandlerGauges(lat, lon, tz, ctx)
	if err != nil {
		errMsg := fmt.Sprintf("cannot create gauges: %v", err)
		log.Printf("Requesting %d,%d at %s failed: %s", lat, lon, tz, errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(latGauge)
	registry.MustRegister(lonGauge)

	promHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	promHandler.ServeHTTP(w, r)
}

func main() {
	listenAddr := flag.String("listen", ":9426", "Listen address to be bound to")
	flag.Parse()

	log.Printf("Starting geohashing_exporter on %s", *listenAddr)

	http.HandleFunc("/metrics", metricsHandler)
	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		log.Panic(err)
	}
}
