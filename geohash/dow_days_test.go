// SPDX-FileCopyrightText: 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package geohash

import (
	"fmt"
	"testing"
	"time"
)

func TestDowHourCheckMarketClosed(t *testing.T) {
	locNy := nyseTz()
	locBerlin, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		ts     string
		loc    *time.Location
		closed bool
	}{
		{"09:30", locNy, false}, // NYSE opening time in NY
		{"09:29", locNy, true},  // before NYSE opening time in NY
		{"09:31", locNy, false}, // after NYSE opening time in NY

		{"09:30", locBerlin, true},  // Berlin's 09:30 is too early in NY.
		{"00:00", locBerlin, false}, // Berlin's midnight is the previous day, still open.
		{"18:00", locBerlin, false}, // Berlin's 18:00 is late enough to be opened in NY.
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s;%v", test.loc, test.ts), func(t *testing.T) {
			date, err := time.ParseInLocation("2006-01-02 15:04", "2022-01-01 "+test.ts, test.loc)
			if err != nil {
				t.Fatal(err)
			}

			closed := dowHourCheckMarketClosed(date)
			if test.closed != closed {
				t.Fatalf("closed should be %t but is %t", test.closed, closed)
			}
		})
	}
}

func TestCorrectDowDate(t *testing.T) {
	tests := []struct {
		date      string
		corrected string
	}{
		// Valid day
		{"2022-01-03", "2022-01-03"},

		// Weekend
		{"2022-01-08", "2022-01-07"}, // Saturday
		{"2022-01-09", "2022-01-07"}, // Sunday

		// https://geohashing.site/geohashing/Dow_holiday#Official_Holidays

		// Fixed holidays
		{"2022-01-01", "2021-12-31"}, // New Year's Day, 2022
		{"2023-01-01", "2022-12-30"}, // New Year's Day, 2023
		{"2022-06-20", "2022-06-17"}, // Juneteenth National Independence Day, 2022
		{"2023-06-19", "2023-06-16"}, // Juneteenth National Independence Day, 2023
		{"2022-07-04", "2022-07-01"}, // Independence Day, 2022
		{"2023-07-04", "2023-07-03"}, // Independence Day, 2023
		{"2022-12-26", "2022-12-23"}, // Christmas, 2022
		{"2023-12-25", "2023-12-22"}, // Christmas, 2023

		// Holidays on the nth weekday of a certain month
		{"2022-01-17", "2022-01-14"}, // Martin Luther King, Jr. Day, 2022
		{"2023-01-16", "2023-01-13"}, // Martin Luther King, Jr. Day, 2023
		{"2022-02-21", "2022-02-18"}, // Washington's Birthday, 2022
		{"2023-02-20", "2023-02-17"}, // Washington's Birthday, 2023
		{"2022-09-05", "2022-09-02"}, // Labor Day, 2022
		{"2023-09-04", "2023-09-01"}, // Labor Day, 2023
		{"2022-11-24", "2022-11-23"}, // Thanksgiving Day, 2022
		{"2023-11-23", "2023-11-22"}, // Thanksgiving Day, 2023

		// Memorial Day
		{"2022-05-30", "2022-05-27"}, // Memorial Day, 2022
		{"2023-05-29", "2023-05-26"}, // Memorial Day, 2023

		// Good Friday
		{"2022-04-15", "2022-04-14"}, // Good Friday, 2022
		{"2023-04-07", "2023-04-06"}, // Good Friday, 2023
	}

	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			date, _ := time.ParseInLocation("2006-01-02 15:04", test.date+" 09:30", nyseTz())
			corrected, _ := time.ParseInLocation("2006-01-02 15:04", test.corrected+" 09:30", nyseTz())

			out, err := correctDowDate(date)
			if err != nil {
				t.Fatal(err)
			}

			if !corrected.Equal(out) {
				t.Fatalf("expected %v instead of %v", corrected, out)
			}
		})
	}
}
