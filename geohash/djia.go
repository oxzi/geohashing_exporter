// SPDX-FileCopyrightText: 2022, 2023 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// This file contains code to determine and cache the Dow Jones Industrial
// Average (DJIA) indicator.

package geohash

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// djiaFetchApi the DJIA for the given date utilizing a given API endpoint.
func djiaFetchApi(apiUrl string, date time.Time, ctx context.Context) (djia float64, err error) {
	reqUrl := date.Format(apiUrl)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return
	}

	if res.StatusCode == http.StatusNotFound {
		err = fmt.Errorf("DJIA API at %q fails with %d, %q", reqUrl, res.StatusCode, body)
		return
	} else if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("DJIA API at %q fails with unexpected status code %d", reqUrl, res.StatusCode)
		return
	}

	djia, err = strconv.ParseFloat(string(body), 64)
	return
}

// djiaFetch the DJIA for the given date utilizing different APIs.
func djiaFetch(date time.Time, ctx context.Context) (djia float64, err error) {
	apiUrls := []string{
		// https://geohashing.site/geohashing/Dow_Jones_Industrial_Average#geo.crox.net_.28recommended.29
		"http://geo.crox.net/djia/2006/01/02",

		// https://geohashing.site/geohashing/Dow_Jones_Industrial_Average#carabiner.peeron.com
		"http://carabiner.peeron.com/xkcd/map/data/2006/01/02",
	}

	type djiaApiResult struct {
		djia float64
		err  error
	}

	subCtx, subCtxCancel := context.WithCancel(ctx)
	defer subCtxCancel()

	results := make(chan djiaApiResult, len(apiUrls))

	for _, apiUrl := range apiUrls {
		go func(apiUrl string) {
			var result djiaApiResult
			result.djia, result.err = djiaFetchApi(apiUrl, date, subCtx)
			results <- result
		}(apiUrl)
	}
	for i := 0; i < len(apiUrls); i++ {
		result := <-results

		if result.err == nil {
			djia = result.djia
			err = nil
			return
		}

		if err == nil {
			err = result.err
		} else {
			err = fmt.Errorf("%v, %w", err, result.err)
		}
	}

	err = fmt.Errorf("Cannot fetch DJIA from any API: %w", err)
	return
}

// dowJonesIndustrialAvgProvider describes an interface which allows both
// querying and caching DJIA values. The only relevant implementation is
// geohash.DowJonesIndustrialAvgCache - use geohashing.NewDjiaCache.
type dowJonesIndustrialAvgProvider interface {
	// Get the Dow Jones Industrial Average (DJIA) for the given date.
	Get(time.Time, context.Context) (float64, error)
}

// DowJonesIndustrialAvgCache implements geohash.dowJonesIndustrialAvgManager
// backed by a LRU cache.
type dowJonesIndustrialAvgCache struct {
	cache *lru.Cache[string, float64]
}

// newDjiaCache to query DJIA with a LRU cache.
func newDjiaCache() (djiaCache *dowJonesIndustrialAvgCache) {
	djiaCache = &dowJonesIndustrialAvgCache{}
	djiaCache.cache, _ = lru.New[string, float64](16)
	return
}

// Get the DJIA value for the given date.
func (djiaCache *dowJonesIndustrialAvgCache) Get(date time.Time, ctx context.Context) (djia float64, err error) {
	cacheKey := date.Format("2006-01-02")
	cachedDjia, cacheHit := djiaCache.cache.Get(cacheKey)
	if cacheHit {
		djia = cachedDjia
		return
	}

	djia, err = djiaFetch(date, ctx)
	if err != nil {
		return
	}

	_ = djiaCache.cache.Add(cacheKey, djia)
	return
}
