// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// This file eases detecting if the New York Stock Exchange (NYSE) was open at
// a given date or if an earlier day should be used - checks weekends and Dow
// holidays. For external usage, there is only the CorrectDowDate function.

package geohash

import (
	"fmt"
	"sync"
	"time"
)

// nyseTz returns the time zone of the NYSE, America/New_York, ET (UTC-05:00)
// with daylight saving time (UTC-04:00).
//
// Please note: This function panics if it is unable to load the time zone.
func nyseTz() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return loc
}

// dowDayValidator is a function mapping a date to a bool, evaluating to true if
// the NYSE is closed at this date.
type dowDayValidator func(time.Time) (isClosed bool)

// dowHourCheckMarketClosed verifies a given time against the NYSE opening time
// in the New York time zone.
func dowHourCheckMarketClosed(date time.Time) bool {
	nyseDate := date.In(nyseTz())
	hour, min, _ := nyseDate.Clock()
	return hour*100+min < 930
}

// dowDayCheckWeekend notifies about closed weekends.
func dowDayCheckWeekend(date time.Time) bool {
	w := date.Weekday()
	return w == time.Saturday || w == time.Sunday
}

// dowYearlyCheck wraps some algorithm™ to calculate a yearly closing day and
// caches this calculation.
type dowYearlyCheck struct {
	algorithm func(int) time.Time
	cache     sync.Map // map[int]time.Time
}

// check if the given day is the free day in its year.
func (yearly *dowYearlyCheck) check(date time.Time) bool {
	yearlyDate, ok := yearly.cache.Load(date.Year())
	if !ok {
		yearlyDate = yearly.algorithm(date.Year())
		yearly.cache.Store(date.Year(), yearlyDate)
	}

	_, thisM, thisD := date.UTC().Date()
	_, freeM, freeD := yearlyDate.(time.Time).UTC().Date()
	return thisM == freeM && thisD == freeD
}

// mkDowYearly creates a dowDayValidator backed by a dowYearlyCheck.
func mkDowYearly(algorithm func(int) time.Time) dowDayValidator {
	yearly := &dowYearlyCheck{
		algorithm: algorithm,
	}
	return yearly.check
}

// mkDowYearlyFixedDate creates a dowDayValidator based on dowYearlyCheck for a
// fixed date, e.g., New Year's Day. However, this takes the US federal law
// (5 U.S.C. 6103) into account and moves holidays from Saturday to Friday and
// from Sunday to Monday.
//
// https://www.opm.gov/policy-data-oversight/pay-leave/federal-holidays/
func mkDowYearlyFixedDate(month time.Month, day int) dowDayValidator {
	return mkDowYearly(func(year int) time.Time {
		day := time.Date(year, month, day, 0, 0, 0, 0, nyseTz())

		switch day.Weekday() {
		case time.Saturday:
			return day.Add(-24 * time.Hour)
		case time.Sunday:
			return day.Add(24 * time.Hour)
		default:
			return day
		}
	})
}

// mkDowYearlyNthDay creates a dowDayValidator based on dowYearlyCheck for
// recurrent events on the nth workday in a month, e.g., Martin Luther King, Jr.
// Day occurring each third Monday in January.
func mkDowYearlyNthDay(month time.Month, nth int, weekday time.Weekday) dowDayValidator {
	return mkDowYearly(func(year int) time.Time {
		day := time.Date(year, month, 1, 0, 0, 0, 0, nyseTz())
		for day.Weekday() != weekday {
			day = day.Add(24 * time.Hour)
		}
		return day.Add(time.Duration(nth-1) * 7 * 24 * time.Hour)
	})
}

// Fixed holidays
var (
	// dowDayNewYearsDay checks for the New Year's Day.
	dowDayNewYearsDay = mkDowYearlyFixedDate(time.January, 1)
	// dowDayJuneteenth checks for the Juneteenth National Independence Day.
	dowDayJuneteenth = mkDowYearlyFixedDate(time.June, 19)
	// dowDayIndependence checks for the Independence Day.
	dowDayIndependence = mkDowYearlyFixedDate(time.July, 4)
	// dowDayChristmas checks for Christmas.
	dowDayChristmas = mkDowYearlyFixedDate(time.December, 25)
)

// Holidays on the nth weekday of a certain month
var (
	// dowDayMartinLutherKingJr checks for Martin Luther King, Jr. Day.
	dowDayMartinLutherKingJr = mkDowYearlyNthDay(time.January, 3, time.Monday)
	// dowDayWashingtonBday checks for Washington's Birthday.
	dowDayWashingtonBday = mkDowYearlyNthDay(time.February, 3, time.Monday)
	// dowDayLaborDay checks for the Labor Day.
	dowDayLaborDay = mkDowYearlyNthDay(time.September, 1, time.Monday)
	// dowDayThanksgivingDay checks for the Thanksgiving Day.
	dowDayThanksgivingDay = mkDowYearlyNthDay(time.November, 4, time.Thursday)
)

// dowDayMemorialDay checks for the Memorial Day, last Monday in May.
var dowDayMemorialDay = mkDowYearly(func(year int) time.Time {
	day := time.Date(year, time.May, 31, 0, 0, 0, 0, nyseTz())
	for day.Weekday() != time.Monday {
		day = day.Add(-24 * time.Hour)
	}
	return day
})

// dowDayGoodFriday checks for the Good Friday based on Gauss' Easter Algorithm.
//
// https://en.wikipedia.org/wiki/Date_of_Easter#Gauss's_Easter_algorithm
var dowDayGoodFriday = mkDowYearly(func(year int) time.Time {
	a := year % 19
	b := year % 4
	c := year % 7
	k := year / 100
	p := (13 + 8*k) / 25
	q := k / 4
	m := (15 - p + k - q) % 30
	n := (4 + k - q) % 7
	d := (19*a + m) % 30
	e := (2*b + 4*c + 6*d + n) % 7

	goodFriday := 20 + d + e
	if goodFriday <= 31 {
		return time.Date(year, time.March, goodFriday, 0, 0, 0, 0, nyseTz())
	} else {
		return time.Date(year, time.April, goodFriday-31, 0, 0, 0, 0, nyseTz())
	}
})

// allDowDayValidators defined above, based on
// https://geohashing.site/geohashing/Dow_holiday#Official_Holidays
var allDowDayValidators = []dowDayValidator{
	dowDayCheckWeekend,

	dowDayNewYearsDay,
	dowDayJuneteenth,
	dowDayIndependence,
	dowDayChristmas,

	dowDayMartinLutherKingJr,
	dowDayWashingtonBday,
	dowDayLaborDay,
	dowDayThanksgivingDay,

	dowDayMemorialDay,
	dowDayGoodFriday,
}

// correctDowDate adjusts a date with regard to the NYSE opening times. Both
// weekends as well as holidays are being checked.
//
// https://geohashing.site/geohashing/Dow_holiday#Official_Holidays
func correctDowDate(date time.Time) (realDate time.Time, err error) {
	realDate = date

	// If the NYSE is not opened yet, jump back to the previous day. However, we
	// cannot subtract 24 hours and handle this check as the other ones, as the
	// following day would also be too early. As this check operates on the time
	// and not the date, it needs to be handled separately.
	if dowHourCheckMarketClosed(realDate) {
		realDate = realDate.Add(-12 * time.Hour)
	}

	for i := 0; i < 7; i++ {
		skip := false
		for _, validator := range allDowDayValidators {
			if validator(realDate) {
				skip = true
				break
			}
		}

		if !skip {
			return
		}
		realDate = realDate.Add(-24 * time.Hour)
	}

	err = fmt.Errorf("cannot correct date: NYSE shouldn't be closed seven days in a row")
	return
}
