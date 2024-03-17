// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package shape

import (
	"github.com/patrickbr/gtfsparser/gtfs"
	"math"
	"strings"
)

// AggrShape is a trip-aggregated shapes containing
// gtfs.Route and gtfs.Trip objects sharing the
// same shape
type AggrShape struct {
	Shape                     *gtfs.Shape
	From                      float64
	To                        float64
	Trips                     map[string]*gtfs.Trip
	Routes                    map[string]*gtfs.Route
	RouteTripCount            map[*gtfs.Route]int
	MeterLength               float64
	NumStops                  map[*gtfs.Route]int
	WheelchairAccessibleTrips map[*gtfs.Route]int
	WheelchairAccessibleStops map[*gtfs.Route]int
}

// NewAggrShape returns a new AggrShape instance
func NewAggrShape() *AggrShape {
	p := AggrShape{
		From:                      math.NaN(),
		To:                        math.NaN(),
		Trips:                     make(map[string]*gtfs.Trip),
		Routes:                    make(map[string]*gtfs.Route),
		RouteTripCount:            make(map[*gtfs.Route]int),
		MeterLength:               0,
		NumStops:                  make(map[*gtfs.Route]int),
		WheelchairAccessibleTrips: make(map[*gtfs.Route]int),
		WheelchairAccessibleStops: make(map[*gtfs.Route]int),
	}
	return &p
}

// GetTripIdsString returns a comma separated list of
// trip IDs contained in this AggrShape
func (as *AggrShape) GetTripIdsString() string {
	keys := make([]string, 0, len(as.Trips))
	for k := range as.Trips {
		keys = append(keys, k)
	}

	return strings.Join(keys, ",")
}

// GetRouteIdsString returns a comma separated list of
// route IDs contained in this AggrShape
func (as *AggrShape) GetRouteIdsString() string {
	keys := make(map[string]struct{})
	for k := range as.Routes {
		keys[k] = struct{}{}
	}

	ids := make([]string, 0)
	for k := range keys {
		ids = append(ids, k)
	}

	return strings.Join(ids, ",")
}

func (as *AggrShape) CalcMeterLength() {
	first := 0
	last := len(as.Shape.Points) - 1

	haveFirst := false

	if !math.IsNaN(as.From) && !math.IsNaN(as.To) {
		for i := 0; i < len(as.Shape.Points); i++ {
			if math.IsNaN(float64(as.Shape.Points[i].Dist_traveled)) {
				first = 0
				last = len(as.Shape.Points) - 1
				break
			}

			if !haveFirst && float64(as.Shape.Points[i].Dist_traveled) >= as.From {
				first = i
				haveFirst = true
			}

			if haveFirst && float64(as.Shape.Points[i].Dist_traveled) > as.To {
				last = i - 1
				break
			}
		}
	}

	mlen := 0.0

	if first > 0 {
		latdiff := as.Shape.Points[first].Lat - as.Shape.Points[first-1].Lat
		londiff := as.Shape.Points[first].Lon - as.Shape.Points[first-1].Lon

		dMeasure := as.Shape.Points[first].Dist_traveled - as.Shape.Points[first-1].Dist_traveled

		lat := as.Shape.Points[first-1].Lat + latdiff/dMeasure*(float32(as.From)-as.Shape.Points[first-1].Dist_traveled)
		lon := as.Shape.Points[first-1].Lon + londiff/dMeasure*(float32(as.From)-as.Shape.Points[first-1].Dist_traveled)

		mlen += haversine(float64(lat), float64(lon), float64(as.Shape.Points[first].Lat), float64(as.Shape.Points[first].Lon))
	}

	if last < len(as.Shape.Points)-1 {
		latdiff := as.Shape.Points[last+1].Lat - as.Shape.Points[last].Lat
		londiff := as.Shape.Points[last+1].Lon - as.Shape.Points[last].Lon

		dMeasure := as.Shape.Points[last+1].Dist_traveled - as.Shape.Points[last].Dist_traveled

		lat := as.Shape.Points[last].Lat + latdiff/dMeasure*(float32(as.To)-as.Shape.Points[last].Dist_traveled)
		lon := as.Shape.Points[last].Lon + londiff/dMeasure*(float32(as.To)-as.Shape.Points[last].Dist_traveled)

		mlen += haversine(float64(lat), float64(lon), float64(as.Shape.Points[last].Lat), float64(as.Shape.Points[last].Lon))
	}

	for i := first + 1; i <= last; i++ {
		mlen += haversineP(as.Shape.Points[i-1], as.Shape.Points[i])
	}

	as.MeterLength = mlen
}

// GetShortNamesString returns a comma separated list of
// the short names of the routes contained in this AggrShape
func (as *AggrShape) GetShortNamesString() string {
	sNames := make(map[string]struct{})
	for _, v := range as.Routes {
		sNames[v.Short_name] = struct{}{}
	}

	sNamesSl := make([]string, 0)
	for k := range sNames {
		sNamesSl = append(sNamesSl, k)
	}

	return strings.Join(sNamesSl, ",")
}

// Calculate the distance in meter between two lat,lng pairs
func haversine(latA float64, lonA float64, latB float64, lonB float64) float64 {
	latA = latA * DEG_TO_RAD
	lonA = lonA * DEG_TO_RAD
	latB = latB * DEG_TO_RAD
	lonB = lonB * DEG_TO_RAD

	dlat := latB - latA
	dlon := lonB - lonA

	sindlat := math.Sin(dlat / 2)
	sindlon := math.Sin(dlon / 2)

	a := sindlat*sindlat + math.Cos(latA)*math.Cos(latB)*sindlon*sindlon

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return c * 6378137.0
}

// Calculate the distance between two ShapePoints
func haversineP(a gtfs.ShapePoint, b gtfs.ShapePoint) float64 {
	return haversine(float64(a.Lat), float64(a.Lon), float64(b.Lat), float64(b.Lon))
}
