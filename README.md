<!--
SPDX-FileCopyrightText: 2022 Alvar Penning

SPDX-License-Identifier: GPL-3.0-or-later
-->

# geohashing\_exporter

Prometheus exporter to be alert about nearby xkcd Geohashes.

[![xkcd: Geohashing, xkcd.com/426](https://imgs.xkcd.com/comics/geohashing.png)](https://xkcd.com/426/)


## Running the Prometheus Exporter

The only real requirement is the Go programming language in version 1.17 or newer.
Maybe older versions will work as well.
Or newer versions won't.
I just had 1.17 installed.

You might also wanna have Prometheus running somewhere, as otherwise it is somehow useless.

```
$ go build ./cmd/geohashing_exporter
$ ./geohashing_exporter
```

## Using the Prometheus Exporter

For a test drive, the exporter can be `curl`ed.
In this example, the coordinates for the coordinate window (50, 8), their neighboring windows, and the Globalhash will be queried.

```
$ export LAT=50 LON=8 TZ=Europe/Berlin
$ curl "http://localhost:9426/metrics?lat=${LAT}&lon=${LON}&tz=${TZ}"
# HELP geohashing_lat Latitude of the geohash.
# TYPE geohashing_lat gauge
geohashing_lat{day_offset="0",location="center"} 50.67375165844576
[…]
# HELP geohashing_lon Longitude of the geohash.
# TYPE geohashing_lon gauge
geohashing_lon{day_offset="0",location="center"} 8.193304942971343
[…]
```

There are only two metrics: `geohashing_lat` and `geohashing_lon` representing the GPS latitude and longitude of a Geohash.

More information is passed through the labels:

* `day_offset` is an integer indicating when those coordinates are valid, where `0` is today, `1` is tomorrow and so on.
  Future days might be available when the New York Stock Exchange (NYSE) will be closed - either due to weekends or [holidays](https://geohashing.site/geohashing/Dow_holiday).
* `location` specifies where the Geohash is located with respect to the requested window:
  * `center` is the Geohash within the requested window,
  * `nw`, `n`, `ne`, `w`, `e`, `sw`, `s`, and `se` describes the Geohash in the coordinate windows northwest, north, …, and southeast of the requested window, and
  * `global` is the unique [Globalhash](https://geohashing.site/geohashing/Globalhash) independent of the requested coordinates.

Btw, in the new world and everywhere west of the longitude -30 there might be no Geohash available between midnight and the NYSE's opening, in New York time.
This is called the [30W Time Zone Rule](https://geohashing.site/geohashing/30W_Time_Zone_Rule) or sometimes _W30_ as I oppose consistency.

Finally, you can configure a [`scrape_config`](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config) in your Prometheus configuration like the following example.

```yaml
scrape_configs:
  - job_name: "geohashing"
    params:
      lat: ["50"]
      lon: ["8"]
      tz: ["Europe/Berlin"]
    static_configs:
      - targets: ["localhost:9426"]
```


## Generate Prometheus Rules for Alerting

Unfortunately, the PromQL does not enable you to calculate the distance between two GPS coordinates in a straight forward way.
Very sad!

That's why the `contrib/prometheus/rule_gen.py` script allows you transpiles the [Haversine formula](https://en.wikipedia.org/wiki/Haversine_formula) against a known location, e.g., your home.

As an example, let's generate a PromQL queries to be used as an alerting rule `expr` to match Geohashes next to 30km and Globalhashes next to 250km near the Marburg castle.

```
$ ./contrib/prometheus/rule_gen.py 50.810222 8.767017 30
$ ./contrib/prometheus/rule_gen.py --globalhash 50.810222 8.767017 250
```

The final alerting rule can be admired in the [`contrib/prometheus/rules-geohashing.yml`](contrib/prometheus/rules-geohashing.yml) file.
Feel free to use a variant of this as one of your Prometheus `rule_files`.


## Golang Geohashing Library

In the odd case that an over-engineered Go library might be needed for the Geohashing algorithm, it is available in the `geohash` directory.

The documentation can be shown with
```
$ go doc -all ./geohash
```
or in this [web documentation thingy](https://pkg.go.dev/github.com/oxzi/geohashing_exporter/geohash).

Should the license - GNU GPLv3 - be an obstacle for your Geohashing-related startup, I am happy to be contacted to arrange an [industry standard agreement](https://www.sqlite.org/copyright.html).


## Is this all supposed to be a joke?

What isn't?
And even if, would this change a thing?
