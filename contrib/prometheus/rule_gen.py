#!/usr/bin/env python3

# SPDX-FileCopyrightText: 2022 Alvar Penning
#
# SPDX-License-Identifier: GPL-3.0-or-later

import argparse
import math


def create_query(lat, lon, distance, unit, globalhash):
    "Generate a Haversine formular implementation in PromQL against a static position."

    d2r = math.pi / 180.0
    square = lambda x: f"(({x}) * ({x}))"

    geohashing_lat = "geohashing_lat" if not globalhash else "geohashing_lat{location=\"global\"}"
    geohashing_lon = "geohashing_lon" if not globalhash else "geohashing_lon{location=\"global\"}"

    dlat = f"(({lat} - {geohashing_lat}) * {d2r})"
    dlon = f"(({lon} - {geohashing_lon}) * {d2r})"

    cLat1 = f"cos({geohashing_lat} * {d2r})"
    cLat2 = math.cos(lat * d2r)

    a = f"{square(f'sin({dlat} / 2.0)')} + {cLat1} * {cLat2} * {square(f'sin({dlon} / 2.0)')}"
    c = f"2 * (sqrt({a}) atan2 sqrt(1-{a}))"

    unit_mult = 6367 if unit == "km" else 3956

    return f"{unit_mult} * {c} <= {distance}"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="geohashing_exporter, Haversine PromQL generator")
    parser.add_argument("lat", type=float, help="Latitude")
    parser.add_argument("lon", type=float, help="Longitude")
    parser.add_argument("distance", type=float, help="Distance")
    parser.add_argument("--unit", choices=["km", "mi"], default="km", help="Unit")
    parser.add_argument("--globalhash", default=False, action="store_true", help="Only globalhashes")

    args = parser.parse_args()

    print(create_query(args.lat, args.lon, args.distance, args.unit, args.globalhash))
