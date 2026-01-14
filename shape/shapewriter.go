// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package shape

import (
	"encoding/csv"
	"fmt"
	"github.com/patrickbr/go-shp"
	"github.com/patrickbr/gtfsparser"
	"github.com/patrickbr/gtfsparser/gtfs"
	"github.com/pebbe/go-proj-4/proj/v5"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var wgs84 = "+proj=longlat +ellps=WGS84 +datum=WGS84 +no_defs"

// ShapeWriter writes shapes to a shapefile
type ShapeWriter struct {
	outProj   *proj.Proj
	wgs84Proj *proj.Proj
	motMap    map[int16]bool
	fldMap    map[string]string
}

type RouteStats struct {
	TotLength float64
	TotFreq   int
}

// NewShapeWriter creates a new ShapeWriter, writing in the specified projection (as proj4 string)
func NewShapeWriter(projection string, motMap map[int16]bool, fldMap map[string]string) *ShapeWriter {
	sw := ShapeWriter{
		motMap: motMap,
		fldMap: fldMap,
	}

	/**
	* NOTE: go-proj-4 does not yet support pj_is_latlong(), which
	* means we have no secure way the test whether the user requested
	* latlng output. If EPSG:4326 is defined in another way than tested
	* here, it will reproject the coordinates to latlng in radians!
	*/
	if projection != "4326" && projection != wgs84 {
		// we need reprojection of coordinates

		var targetProj *proj.Proj
		wgs84, err := proj.NewProj(wgs84)
		if err != nil {
			panic(fmt.Sprintf("Could not init WGS84 projection, maybe proj4 is not available? (%s)", err))
		}

		if _, err := strconv.Atoi(projection); err == nil {
			// srid supplied
			pr, err := proj.NewProj("+init=epsg:" + projection)
			if err != nil {
				panic(fmt.Sprintf("Could not init projection with SRID %s", projection))
			}
			targetProj = pr
		} else {
			// treat as proj4 string
			pr, err := proj.NewProj(projection)
			if err != nil {
				panic(fmt.Sprintf("Could not init projection %s (%s)", projection, err))
			}
			targetProj = pr
		}

		sw.wgs84Proj = wgs84
		sw.outProj = targetProj
	}

	return &sw
}

// WriteShapeExplicit
func (sw *ShapeWriter) WriteShapeExplicit(s *gtfs.Shape, outFile string) int {
	shape, err := shp.Create(sw.getTripShapeFileName(outFile), shp.POLYLINE)

	if err != nil {
		panic(fmt.Sprintf("Could not open shapefile for writing (%s)", err))
	}
	defer shape.Close()

	n := 0

	shape.SetFields(sw.getFieldSizesForShape(s))

	for i := 1; i < len(s.Points); i++ {
		from := math.NaN()
		to := math.NaN()

		points := sw.gtfsShapePointsToShpLinePoints(gtfs.ShapePoints{s.Points[i-1], s.Points[i]}, from, to)
		parts := [][]shp.Point{points}

		shape.Write(shp.NewPolyLine(parts))

		shape.WriteAttribute(n, 0, s.Id)
		shape.WriteAttribute(n, 1, float64(s.Points[i-1].Dist_traveled))
		shape.WriteAttribute(n, 2, float64(s.Points[i].Dist_traveled))
		n = n + 1
	}
	return n
}

// WriteTripsExplicit writes the shapes contained in Feed f to outFile, with each trip as an
// explicit geometry with all trip attributes
func (sw *ShapeWriter) WriteTripsExplicit(f *gtfsparser.Feed, tid string, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileName(outFile), shp.POLYLINE)

	if err != nil {
		panic(fmt.Sprintf("Could not open shapefile for writing (%s)", err))
	}
	defer shape.Close()

	shape.SetFields(sw.getFieldSizesForTrips(f.Trips))

	n := 0
	calcedShapes := make(map[string]*shp.PolyLine)

	trips := f.Trips

	if len(tid) > 0 {
		trip, ok := f.Trips[tid]
		if ok {
			trips = make(map[string]*gtfs.Trip)
			trips[tid] = trip
		} else {
			panic(fmt.Sprintf("Trip not found: %s", tid))
		}
	}

	// iterate through trips
	for _, trip := range trips {
		if len(sw.motMap) > 0 && !sw.motMap[trip.Route.Type] {
			continue
		}

		if len(tid) > 0 {
			if trip.Shape != nil {
				for i := 1; i < len(trip.StopTimes); i++ {
					from := float64(trip.StopTimes[i-1].Shape_dist_traveled())
					to := float64(trip.StopTimes[i].Shape_dist_traveled())

					points := sw.gtfsShapePointsToShpLinePoints(trip.Shape.Points, from, to)
					parts := [][]shp.Point{points}

					shape.Write(shp.NewPolyLine(parts))
				}
			} else {
				for i := 1; i < len(trip.StopTimes); i++ {
					parts := [][]shp.Point{}
					parts[0] = append(parts[0], *sw.gtfsStopToShpPoint(trip.StopTimes[i-1].Stop()))
					parts[0] = append(parts[0], *sw.gtfsStopToShpPoint(trip.StopTimes[i].Stop()))

					shape.Write(shp.NewPolyLine(parts))
				}
			}

			for i := 1; i < len(trip.StopTimes); i++ {
				stCur := trip.StopTimes[i]
				shape.WriteAttribute(n, 0, trip.Id)

				if stCur.Headsign() != nil {
					shape.WriteAttribute(n, 1, *stCur.Headsign())
				} else if trip.Headsign != nil {
					shape.WriteAttribute(n, 1, *trip.Headsign)
				}

				if trip.Short_name != nil {
					shape.WriteAttribute(n, 2, *trip.Short_name)
				} else {
					shape.WriteAttribute(n, 2, "")
				}

				shape.WriteAttribute(n, 3, trip.Direction_id)
				if trip.Block_id != nil {
					shape.WriteAttribute(n, 4, *trip.Block_id)
				}
				shape.WriteAttribute(n, 5, trip.Wheelchair_accessible)
				shape.WriteAttribute(n, 6, trip.Bikes_allowed)
				shape.WriteAttribute(n, 7, trip.Route.Short_name)
				shape.WriteAttribute(n, 8, trip.Route.Long_name)
				shape.WriteAttribute(n, 9, trip.Route.Desc)
				shape.WriteAttribute(n, 10, trip.Route.Type)
				shape.WriteAttribute(n, 11, trip.Route.Url)
				shape.WriteAttribute(n, 12, trip.Route.Color)
				shape.WriteAttribute(n, 13, trip.Route.Text_color)

				n = n + 1
			}
		} else {
			if trip.Shape != nil {
				from := math.NaN()
				to := math.NaN()
				if len(trip.StopTimes) > 0 {
					from = float64(trip.StopTimes[0].Shape_dist_traveled())
					to = float64(trip.StopTimes[len(trip.StopTimes)-1].Shape_dist_traveled())
				}
				points := sw.gtfsShapePointsToShpLinePoints(trip.Shape.Points, from, to)
				parts := [][]shp.Point{points}

				// prevent re-calcing of polylines for each trip
				if val, ok := calcedShapes[trip.Shape.Id]; ok {
					shape.Write(val)
				} else {
					calcedShapes[trip.Shape.Id] = shp.NewPolyLine(parts)
					shape.Write(calcedShapes[trip.Shape.Id])
				}
			} else {
				// use station positions as polyline anchors
				points := sw.gtfsStationPointsToShpLinePoints(trip.StopTimes)
				parts := [][]shp.Point{points}

				shape.Write(shp.NewPolyLine(parts))
			}
			shape.WriteAttribute(n, 0, trip.Id)
			if trip.Headsign != nil {
				shape.WriteAttribute(n, 1, *trip.Headsign)
			}
			if trip.Short_name != nil {
				shape.WriteAttribute(n, 2, *trip.Short_name)
			}
			shape.WriteAttribute(n, 3, trip.Direction_id)
			if trip.Block_id != nil {
				shape.WriteAttribute(n, 4, *trip.Block_id)
			}
			shape.WriteAttribute(n, 5, trip.Wheelchair_accessible)
			shape.WriteAttribute(n, 6, trip.Bikes_allowed)
			shape.WriteAttribute(n, 7, trip.Route.Short_name)
			shape.WriteAttribute(n, 8, trip.Route.Long_name)
			shape.WriteAttribute(n, 9, trip.Route.Desc)
			shape.WriteAttribute(n, 10, trip.Route.Type)
			shape.WriteAttribute(n, 11, trip.Route.Url)
			shape.WriteAttribute(n, 12, trip.Route.Color)
			shape.WriteAttribute(n, 13, trip.Route.Text_color)

			n = n + 1
		}

	}

	return n
}

func (sw *ShapeWriter) WriteRouteOverviewCsv(f *gtfsparser.Feed, typeMap map[int16]string, routeAddFlds []string, outFile string) {
	csvFile, err := os.Create(sw.getCsvFileName(outFile))

	if err != nil {
		panic(fmt.Sprintf("Could not open CSV file for writing (%s)", err))
	}

	csvwriter := csv.NewWriter(csvFile)

	headers := []string{sw.fldName("Route_id"), sw.fldName("Short_name"), sw.fldName("Long_name"), sw.fldName("Type"), sw.fldName("Frequency"), sw.fldName("Km_len"), sw.fldName("Km_tot"), sw.fldName("Km_max"), sw.fldName("Agency_name"), sw.fldName("Agency_url"), sw.fldName("Wchair_tr"), sw.fldName("Wchair_st")}

	for _, field := range routeAddFlds {
		headers = append(headers, sw.fldName(field))
	}

	csvwriter.Write(headers)

	aggrShapes, routeShapes := sw.getAggrShapes(f.Trips, f)

	for route, shapes := range routeShapes {
		vals := []string{route.Id, route.Short_name, route.Long_name}

		if str, ok := typeMap[route.Type]; ok {
			vals = append(vals, str)
		} else {
			vals = append(vals, strconv.FormatInt(int64(route.Type), 10))
		}

		totFreq := 0
		uniqueAggregatedFreq := 0
		totMeterLength := 0.0
		totMeterLengthSingular := 0.0
		maxMeterLength := 0.0
		wheelchairTripsTot := 0
		wheelchairStopsTot := 0
		numStopsTot := 0

		for s, _ := range shapes {
			aggrShp := aggrShapes[s]
			totFreq += aggrShp.RouteTripCount[route]

			uniqueAggregatedFreq += aggrShp.RouteUniqueTripCount[route]

			totMeterLength += aggrShp.MeterLength * float64(aggrShp.RouteTripCount[route])
			totMeterLengthSingular += aggrShp.MeterLength
			if aggrShp.MeterLength > maxMeterLength {
				maxMeterLength = aggrShp.MeterLength
			}
			wheelchairTripsTot += aggrShp.WheelchairAccessibleTrips[route]
			wheelchairStopsTot += aggrShp.WheelchairAccessibleStops[route]
			numStopsTot += aggrShp.NumStops[route]
		}

		vals = append(vals, strconv.FormatInt(int64(uniqueAggregatedFreq), 10))
		vals = append(vals, strconv.FormatFloat(((totMeterLength)/float64(totFreq)) / float64(1000), 'f', 10, 64))
		vals = append(vals, strconv.FormatFloat(totMeterLength / 1000.0, 'f', 10, 64))
		vals = append(vals, strconv.FormatFloat(maxMeterLength / 1000.0, 'f', 10, 64))
		vals = append(vals, route.Agency.Name)
		if route.Agency.Url != nil {
			vals = append(vals, route.Agency.Url.String())
		} else {
			vals = append(vals, "")
		}

		vals = append(vals, strconv.FormatFloat(float64(wheelchairTripsTot)/float64(totFreq), 'f', 10, 64))
		vals = append(vals, strconv.FormatFloat(float64(wheelchairStopsTot)/float64(numStopsTot), 'f', 10, 64))

		for _, field := range routeAddFlds {
			vald := ""
			if vals, ok := f.RoutesAddFlds[field]; ok {
				if val, ok := vals[route.Id]; ok {
					vald = val
				}
			}

			vals = append(vals, vald)
		}

		csvwriter.Write(vals)
	}

	csvwriter.Flush()
	csvFile.Close()
}

func (sw *ShapeWriter) WriteRouteShapes(f *gtfsparser.Feed, typeMap map[int16]string, routeAddFlds []string, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileName(outFile), shp.POLYLINE)

	if err != nil {
		panic(fmt.Sprintf("Could not open shapefile for writing (%s)", err))
	}
	defer shape.Close()

	n := 0

	// get aggreshape map
	// aggrShapes, routeStats := sw.getAggrShapes(f.Trips)
	aggrShapes, _ := sw.getAggrShapes(f.Trips, f)
	shape.SetFields(sw.getFieldSizesForRouteShapes(aggrShapes, typeMap, routeAddFlds, f))

	for _, aggrShape := range aggrShapes {
		points := sw.gtfsShapePointsToShpLinePoints(aggrShape.Shape.Points, aggrShape.From, aggrShape.To)
		parts := [][]shp.Point{points}

		for _, r := range aggrShape.Routes {
			shape.Write(shp.NewPolyLine(parts))

			shape.WriteAttribute(n, 0, r.Id)
			shape.WriteAttribute(n, 1, r.Short_name)
			shape.WriteAttribute(n, 2, r.Long_name)
			if str, ok := typeMap[r.Type]; ok {
				shape.WriteAttribute(n, 3, str)
			} else {
				shape.WriteAttribute(n, 3, strconv.FormatInt(int64(r.Type), 10))
			}

			// number of trips
			shape.WriteAttribute(n, 4, aggrShape.RouteTripCount[r])

			// length in km
			shape.WriteAttribute(n, 5, aggrShape.MeterLength / 1000.0)

			// route tot travelled in km
			shape.WriteAttribute(n, 6, (float64(aggrShape.RouteTripCount[r])*aggrShape.MeterLength) / 1000.0)

			// agency name
			shape.WriteAttribute(n, 7, r.Agency.Name)

			// agency url
			shape.WriteAttribute(n, 8, r.Agency.Url.String())

			// wheelchair trips
			shape.WriteAttribute(n, 9, float64(aggrShape.WheelchairAccessibleTrips[r])/float64(aggrShape.RouteTripCount[r]))

			// wheelchair stops
			shape.WriteAttribute(n, 10, float64(aggrShape.WheelchairAccessibleStops[r])/float64(aggrShape.NumStops[r]))

			i := 11

			for _, field := range routeAddFlds {
				if flds, ok := f.RoutesAddFlds[field]; ok {
					if val, ok := flds[r.Id]; ok {
						shape.WriteAttribute(n, i, val)
					} else {
						shape.WriteAttribute(n, i, "")
					}
				} else {
					shape.WriteAttribute(n, i, "")
				}
				i += 1
			}

			n = n + 1
		}
	}

	return n
}

// WriteShapes writes the shapes contained in Feed f to outFile, with each shape containing
// aggregrated trip/route information
func (sw *ShapeWriter) WriteShapes(f *gtfsparser.Feed, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileName(outFile), shp.POLYLINE)

	if err != nil {
		panic(fmt.Sprintf("Could not open shapefile for writing (%s)", err))
	}
	defer shape.Close()

	n := 0

	// get aggreshape map
	aggrShapes, _ := sw.getAggrShapes(f.Trips, f)
	shape.SetFields(sw.getFieldSizesForShapes(aggrShapes))

	for _, aggrShape := range aggrShapes {
		points := sw.gtfsShapePointsToShpLinePoints(aggrShape.Shape.Points, aggrShape.From, aggrShape.To)
		parts := [][]shp.Point{points}

		shape.Write(shp.NewPolyLine(parts))

		shape.WriteAttribute(n, 0, aggrShape.Shape.Id)
		shape.WriteAttribute(n, 1, aggrShape.GetTripIdsString())
		shape.WriteAttribute(n, 2, aggrShape.GetRouteIdsString())
		shape.WriteAttribute(n, 3, aggrShape.GetShortNamesString())

		n = n + 1
	}

	return n
}

// WriteTripStops writes the trip stops for a single trip
func (sw *ShapeWriter) WriteTripStops(f *gtfsparser.Feed, tid string, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileNameTripStops(outFile), shp.POINT)

	if err != nil {
		panic(fmt.Sprintf("Could not open shapefile for writing (%s)", err))
	}
	defer shape.Close()

	t, ok := f.Trips[tid]

	if !ok {
		panic(fmt.Sprintf("Trip not found:", tid))
	}

	n := 0

	// get aggreshape map
	shape.SetFields(sw.getFieldSizesForTripStops(t))

	for _, st := range t.StopTimes {
		stop := st.Stop()
		point := sw.gtfsStopToShpPoint(stop)

		shape.Write(point)

		shape.WriteAttribute(n, 0, stop.Id)
		shape.WriteAttribute(n, 1, stop.Code)
		shape.WriteAttribute(n, 2, stop.Name)
		shape.WriteAttribute(n, 3, stop.Desc)
		shape.WriteAttribute(n, 4, stop.Zone_id)
		shape.WriteAttribute(n, 5, stop.Url)
		shape.WriteAttribute(n, 6, stop.Location_type)
		shape.WriteAttribute(n, 7, stop.Parent_station)
		shape.WriteAttribute(n, 8, stop.Timezone)
		wb := int32(stop.Wheelchair_boarding)
		shape.WriteAttribute(n, 9, wb)
		seq := st.Sequence()
		shape.WriteAttribute(n, 10, seq)
		shape.WriteAttribute(n, 11, fmt.Sprintf("%02d:%02d:%02d", st.Arrival_time().Hour, st.Arrival_time().Minute, st.Arrival_time().Second))
		shape.WriteAttribute(n, 12, fmt.Sprintf("%02d:%02d:%02d", st.Departure_time().Hour, st.Departure_time().Minute, st.Departure_time().Second))
		shape.WriteAttribute(n, 13, float64(st.Shape_dist_traveled()))

		n = n + 1
	}

	return n
}

// WriteStops writes the stations contained in Feed f to outFile
func (sw *ShapeWriter) WriteStops(f *gtfsparser.Feed, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileNameStations(outFile), shp.POINT)

	if err != nil {
		panic(fmt.Sprintf("Could not open shapefile for writing (%s)", err))
	}
	defer shape.Close()

	n := 0

	// get aggreshape map
	shape.SetFields(sw.getFieldSizesForStops(f.Stops))

	for _, stop := range f.Stops {
		point := sw.gtfsStopToShpPoint(stop)

		shape.Write(point)

		shape.WriteAttribute(n, 0, stop.Id)
		shape.WriteAttribute(n, 1, stop.Code)
		shape.WriteAttribute(n, 2, stop.Name)
		shape.WriteAttribute(n, 3, stop.Desc)
		shape.WriteAttribute(n, 4, stop.Zone_id)
		shape.WriteAttribute(n, 5, stop.Url)
		shape.WriteAttribute(n, 6, stop.Location_type)
		shape.WriteAttribute(n, 7, stop.Parent_station)
		shape.WriteAttribute(n, 8, stop.Timezone)
		shape.WriteAttribute(n, 9, stop.Wheelchair_boarding)

		n = n + 1
	}

	return n
}

// return aggregrated shapes from GTFS trips
func (sw *ShapeWriter) getAggrShapes(trips map[string]*gtfs.Trip, feed *gtfsparser.Feed) (map[string]*AggrShape, map[*gtfs.Route]map[string]bool) {
	ret := make(map[string]*AggrShape)
	routeShapes := make(map[*gtfs.Route]map[string]bool)

	// iterate through all trips
	for _, trip := range trips {
		if trip.Shape == nil || (len(sw.motMap) > 0 && !sw.motMap[trip.Route.Type]) || len(trip.StopTimes) < 2 {
			continue
		}

		numOnOffStops := 0

		for _, st := range trip.StopTimes {
			if st.Drop_off_type() != 1 || st.Pickup_type() != 1 {
				numOnOffStops += 1
			}
		}

		aggrShapeId := trip.Shape.Id

		if trip.StopTimes[0].HasDistanceTraveled() && trip.StopTimes[len(trip.StopTimes)-1].HasDistanceTraveled() {
			from := strconv.FormatFloat(float64(trip.StopTimes[0].Shape_dist_traveled()), 'f', 1, 64)
			to := strconv.FormatFloat(float64(trip.StopTimes[len(trip.StopTimes)-1].Shape_dist_traveled()), 'f', 1, 64)
			aggrShapeId += "%%%%%" + from + ":" + to
		}

		if _, ok := routeShapes[trip.Route]; !ok {
			routeShapes[trip.Route] = make(map[string]bool)
		}

		routeShapes[trip.Route][aggrShapeId] = true

		// check if shape is already present
		if _, ok := ret[aggrShapeId]; !ok {
			ret[aggrShapeId] = NewAggrShape()
			ret[aggrShapeId].Shape = trip.Shape
			ret[aggrShapeId].From = float64(trip.StopTimes[0].Shape_dist_traveled())
			ret[aggrShapeId].To = float64(trip.StopTimes[len(trip.StopTimes)-1].Shape_dist_traveled())

			ret[aggrShapeId].CalcMeterLength()
		}

		ret[aggrShapeId].Trips[trip.Id] = trip
		ret[aggrShapeId].Routes[trip.Route.Id] = trip.Route

		if _, ok := ret[aggrShapeId].WheelchairAccessibleTrips[trip.Route]; !ok {
			ret[aggrShapeId].WheelchairAccessibleTrips[trip.Route] = 0
		}

		if _, ok := ret[aggrShapeId].WheelchairAccessibleStops[trip.Route]; !ok {
			ret[aggrShapeId].WheelchairAccessibleStops[trip.Route] = 0
		}

		if _, ok := ret[aggrShapeId].NumStops[trip.Route]; !ok {
			ret[aggrShapeId].NumStops[trip.Route] = 0
		}

		if _, ok := ret[aggrShapeId].RouteTripCount[trip.Route]; !ok {
			ret[aggrShapeId].RouteTripCount[trip.Route] = 0
		}

		start := trip.Service.GetFirstActiveDate()
		end := trip.Service.GetLastActiveDate()
		endT := end.GetTime()

		for d := start; !d.GetTime().After(endT); d = d.GetOffsettedDate(1) {
			if trip.Service.IsActiveOn(d) {
				ret[aggrShapeId].RouteTripCount[trip.Route] += 1

				vals, ok := feed.TripsAddFlds["__trip_count_no_count"]
				if ok {
					val, ok := vals[trip.Id]
					if !ok || val != "1" {
						ret[aggrShapeId].RouteUniqueTripCount[trip.Route] += 1
					}
				} else {
					ret[aggrShapeId].RouteUniqueTripCount[trip.Route] += 1
				}

				ret[aggrShapeId].NumStops[trip.Route] += numOnOffStops

				if trip.Wheelchair_accessible == 1 {
					ret[aggrShapeId].WheelchairAccessibleTrips[trip.Route] += 1
				}

				for _, st := range trip.StopTimes {
					if st.Stop().Wheelchair_boarding == 1 || (st.Stop().Parent_station != nil && st.Stop().Parent_station.Wheelchair_boarding == 1) {
						ret[aggrShapeId].WheelchairAccessibleStops[trip.Route] += 1
					}
				}
			}
		}
	}

	return ret, routeShapes
}

// returns a shapefile geometry from a GTFS shape, reprojected
func (sw *ShapeWriter) gtfsShapePointsToShpLinePoints(gtfsshape gtfs.ShapePoints, from float64, to float64) []shp.Point {
	first := 0
	last := len(gtfsshape) - 1

	haveFirst := false

	ret := make([]shp.Point, 0)

	if !math.IsNaN(from) && !math.IsNaN(to) {
		for i := 0; i < len(gtfsshape); i++ {
			if math.IsNaN(float64(gtfsshape[i].Dist_traveled)) {
				first = 0
				last = len(gtfsshape) - 1
				break
			}

			if !haveFirst && float64(gtfsshape[i].Dist_traveled) >= from {
				first = i
				haveFirst = true
			}

			if haveFirst && float64(gtfsshape[i].Dist_traveled) > to {
				last = i - 1
				break
			}
		}
	}

	if first > 0 {
		latdiff := float64(gtfsshape[first].Lat) - float64(gtfsshape[first-1].Lat)
		londiff := float64(gtfsshape[first].Lon) - float64(gtfsshape[first-1].Lon)

		dMeasure := float64(gtfsshape[first].Dist_traveled) - float64(gtfsshape[first-1].Dist_traveled)

		lat := float64(gtfsshape[first-1].Lat) + latdiff/dMeasure*((from)-float64(gtfsshape[first-1].Dist_traveled))
		lon := float64(gtfsshape[first-1].Lon) + londiff/dMeasure*((from)-float64(gtfsshape[first-1].Dist_traveled))

		if sw.outProj != nil {
			x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(lon)), proj.DegToRad(float64(lat)))
			ret = append(ret, shp.Point{x, y})
		} else {
			ret = append(ret, shp.Point{float64(lon), float64(lat)})
		}
	}

	for i := first; i <= last; i++ {
		if sw.outProj != nil {
			x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(gtfsshape[i].Lon)), proj.DegToRad(float64(gtfsshape[i].Lat)))
			ret = append(ret, shp.Point{x, y})
		} else {
			ret = append(ret, shp.Point{float64(gtfsshape[i].Lon), float64(gtfsshape[i].Lat)})
		}
	}

	if last < len(gtfsshape)-1 {
		latdiff := float64(gtfsshape[last+1].Lat) - float64(gtfsshape[last].Lat)
		londiff := float64(gtfsshape[last+1].Lon) - float64(gtfsshape[last].Lon)

		dMeasure := float64(gtfsshape[last+1].Dist_traveled) - float64(gtfsshape[last].Dist_traveled)

		lat := float64(gtfsshape[last].Lat) + latdiff/dMeasure*((to)-float64(gtfsshape[last].Dist_traveled))
		lon := float64(gtfsshape[last].Lon) + londiff/dMeasure*((to)-float64(gtfsshape[last].Dist_traveled))

		if sw.outProj != nil {
			x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(lon)), proj.DegToRad(float64(lat)))
			ret = append(ret, shp.Point{x, y})
		} else {
			ret = append(ret, shp.Point{float64(lon), float64(lat)})
		}
	}

	return ret
}

// returns a shapefile geometry from a GTFS shape, reprojected
func (sw *ShapeWriter) gtfsStopToShpPoint(stop *gtfs.Stop) *shp.Point {
	if sw.outProj != nil {
		x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(stop.Lon)), proj.DegToRad(float64(stop.Lat)))
		return &shp.Point{X: x, Y: y}
	}
	return &shp.Point{X: float64(stop.Lon), Y: float64(stop.Lat)}
}

/**
* Returns a shapefile geometry from a GTFS station list (if shapes are not available in the feed), reprojected
*/
func (sw *ShapeWriter) gtfsStationPointsToShpLinePoints(stoptimes gtfs.StopTimes) []shp.Point {
	ret := make([]shp.Point, len(stoptimes))
	for i, st := range stoptimes {
		if sw.outProj != nil {
			x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(st.Stop().Lon)), proj.DegToRad(float64(st.Stop().Lat)))
			ret[i].Y = y
			ret[i].X = x
		} else {
			ret[i].Y = float64(st.Stop().Lat)
			ret[i].X = float64(st.Stop().Lon)
		}
	}

	return ret
}

func (sw *ShapeWriter) getFieldSizesForShape(shape *gtfs.Shape) []shp.Field {
	idSize := uint8(0)

	if uint8(min(254, len(shape.Id))) > idSize {
		idSize = uint8(min(254, len(shape.Id)))
	}

	return []shp.Field{
		shp.StringField(sw.fldName("Id"), idSize),
		shp.FloatField(sw.fldName("DistTravFrom"), 64,10),
		shp.FloatField(sw.fldName("DistTravTo"), 64,10),
	}
}

/**
* Calculate the optimal shapefile attribute field sizes to hold trip stop attributes
*/
func (sw *ShapeWriter) getFieldSizesForTripStops(trip *gtfs.Trip) []shp.Field {
	idSize := uint8(0)
	codeSize := uint8(0)
	nameSize := uint8(0)
	descSize := uint8(0)
	zoneIDSize := uint8(0)
	urlSize := uint8(0)
	parentStationSize := uint8(0)
	timezoneSize := uint8(0)

	for _, stopTimes := range trip.StopTimes {
		st := stopTimes.Stop()
		if uint8(min(254, len(st.Id))) > idSize {
			idSize = uint8(min(254, len(st.Id)))
		}
		if uint8(min(254, len(st.Code))) > codeSize {
			codeSize = uint8(min(254, len(st.Code)))
		}
		if uint8(min(254, len(st.Name))) > nameSize {
			nameSize = uint8(min(254, len(st.Name)))
		}
		if uint8(min(254, len(st.Desc))) > descSize {
			descSize = uint8(min(254, len(st.Desc)))
		}
		if uint8(min(254, len(st.Zone_id))) > zoneIDSize {
			zoneIDSize = uint8(min(254, len(st.Zone_id)))
		}
		if st.Url != nil && uint8(min(254, len(st.Url.String()))) > urlSize {
			urlSize = uint8(min(254, len(st.Url.String())))
		}
		if st.Parent_station != nil && uint8(min(254, len(st.Parent_station.Id))) > parentStationSize {
			parentStationSize = uint8(min(254, len(st.Parent_station.Id)))
		}
		if uint8(min(254, len(st.Timezone.GetTzString()))) > timezoneSize {
			timezoneSize = uint8(min(254, len(st.Timezone.GetTzString())))
		}
	}

	return []shp.Field{
		shp.StringField(sw.fldName("Id"), idSize),
		shp.StringField(sw.fldName("Code"), codeSize),
		shp.StringField(sw.fldName("Name"), nameSize),
		shp.StringField(sw.fldName("Desc"), descSize),
		shp.StringField(sw.fldName("Zone_id"), zoneIDSize),
		shp.StringField(sw.fldName("Url"), urlSize),
		shp.NumberField(sw.fldName("Location_type"), 1),
		shp.StringField(sw.fldName("Parent_station"), parentStationSize),
		shp.StringField(sw.fldName("Timezone"), timezoneSize),
		shp.NumberField(sw.fldName("Wheelchair_boarding"), 32),
		shp.NumberField(sw.fldName("Sequence"), 32),
		shp.StringField(sw.fldName("Arrival"), 8),
		shp.StringField(sw.fldName("Departure"), 8),
		shp.FloatField(sw.fldName("ShapeDistTraveled"), 64, 10),
	}
}

/**
* Calculate the optimal shapefile attribute field sizes to hold stop attributes
*/
func (sw *ShapeWriter) getFieldSizesForStops(stops map[string]*gtfs.Stop) []shp.Field {
	idSize := uint8(0)
	codeSize := uint8(0)
	nameSize := uint8(0)
	descSize := uint8(0)
	zoneIDSize := uint8(0)
	urlSize := uint8(0)
	parentStationSize := uint8(0)
	timezoneSize := uint8(0)

	for _, st := range stops {
		if uint8(min(254, len(st.Id))) > idSize {
			idSize = uint8(min(254, len(st.Id)))
		}
		if uint8(min(254, len(st.Code))) > codeSize {
			codeSize = uint8(min(254, len(st.Code)))
		}
		if uint8(min(254, len(st.Name))) > nameSize {
			nameSize = uint8(min(254, len(st.Name)))
		}
		if uint8(min(254, len(st.Desc))) > descSize {
			descSize = uint8(min(254, len(st.Desc)))
		}
		if uint8(min(254, len(st.Zone_id))) > zoneIDSize {
			zoneIDSize = uint8(min(254, len(st.Zone_id)))
		}
		if st.Url != nil && uint8(min(254, len(st.Url.String()))) > urlSize {
			urlSize = uint8(min(254, len(st.Url.String())))
		}
		if st.Parent_station != nil && uint8(min(254, len(st.Parent_station.Id))) > parentStationSize {
			parentStationSize = uint8(min(254, len(st.Parent_station.Id)))
		}
		if uint8(min(254, len(st.Timezone.GetTzString()))) > timezoneSize {
			timezoneSize = uint8(min(254, len(st.Timezone.GetTzString())))
		}
	}

	return []shp.Field{
		shp.StringField(sw.fldName("Id"), idSize),
		shp.StringField(sw.fldName("Code"), codeSize),
		shp.StringField(sw.fldName("Name"), nameSize),
		shp.StringField(sw.fldName("Desc"), descSize),
		shp.StringField(sw.fldName("Zone_id"), zoneIDSize),
		shp.StringField(sw.fldName("Url"), urlSize),
		shp.NumberField(sw.fldName("Location_type"), 1),
		shp.StringField(sw.fldName("Parent_station"), parentStationSize),
		shp.StringField(sw.fldName("Timezone"), timezoneSize),
		shp.NumberField(sw.fldName("Wheelchair_boarding"), 32),
	}
}

/**
* Calculate the optimal shapefile attribute field sizes to hold trip/route fields
*/
func (sw *ShapeWriter) getFieldSizesForTrips(trips map[string]*gtfs.Trip) []shp.Field {
	idSize := uint8(0)
	headsignSize := uint8(0)
	shortNameSize := uint8(0)
	blockIDSize := uint8(0)
	rShortNameSize := uint8(0)
	rLongNameSize := uint8(0)
	rDescSize := uint8(0)
	rURLSize := uint8(0)
	rColorSize := uint8(0)
	rTextColorSize := uint8(0)

	for _, st := range trips {
		if uint8(min(254, len(st.Id))) > idSize {
			idSize = uint8(min(254, len(st.Id)))
		}
		if uint8(min(254, len(*st.Headsign))) > headsignSize {
			headsignSize = uint8(min(254, len(*st.Headsign)))
		}
		if st.Short_name != nil && uint8(min(254, len(*st.Short_name))) > shortNameSize {
			shortNameSize = uint8(min(254, len(*st.Short_name)))
		}
		if st.Block_id != nil && uint8(min(254, len(*st.Block_id))) > blockIDSize {
			blockIDSize = uint8(min(254, len(*st.Block_id)))
		}
		if uint8(min(254, len(st.Route.Short_name))) > rShortNameSize {
			rShortNameSize = uint8(min(254, len(st.Route.Short_name)))
		}
		if uint8(min(254, len(st.Route.Long_name))) > rLongNameSize {
			rLongNameSize = uint8(min(254, len(st.Route.Long_name)))
		}
		if uint8(min(254, len(st.Route.Desc))) > rDescSize {
			rDescSize = uint8(min(254, len(st.Route.Desc)))
		}
		if st.Route.Url != nil && (uint8(min(254, len(st.Route.Url.String()))) > rURLSize) {
			rURLSize = uint8(min(254, len(st.Route.Url.String())))
		}
		if uint8(min(254, len(st.Route.Color))) > rColorSize {
			rColorSize = uint8(min(254, len(st.Route.Color)))
		}
		if uint8(min(254, len(st.Route.Text_color))) > rTextColorSize {
			rTextColorSize = uint8(min(254, len(st.Route.Text_color)))
		}
	}

	return []shp.Field{
		shp.StringField(sw.fldName("Id"), idSize),
		shp.StringField(sw.fldName("Headsign"), headsignSize),
		shp.StringField(sw.fldName("ShortName"), shortNameSize),
		shp.NumberField(sw.fldName("Dir_id"), 16),
		shp.StringField(sw.fldName("BlockId"), blockIDSize),
		shp.NumberField(sw.fldName("Wheelchr_a"), 1),
		shp.NumberField(sw.fldName("Bikes_alwd"), 1),
		shp.StringField(sw.fldName("R_ShrtName"), rShortNameSize),
		shp.StringField(sw.fldName("R_LongName"), rLongNameSize),
		shp.StringField(sw.fldName("R_Desc"), rDescSize),
		shp.NumberField(sw.fldName("R_Type"), 16),
		shp.StringField(sw.fldName("R_URL"), rURLSize),
		shp.StringField(sw.fldName("R_Color"), rColorSize),
		shp.StringField(sw.fldName("R_TextColor"), rTextColorSize),
	}
}

/**
* Calculate the optimal shapefile attribute field sizes to hold aggregated trip/route fields
*/
func (sw *ShapeWriter) getFieldSizesForShapes(shapes map[string]*AggrShape) []shp.Field {
	idSize := uint8(0)
	tIdsSize := uint8(0)
	rIdsSize := uint8(0)
	rShortNamesSize := uint8(0)

	for _, s := range shapes {
		if uint8(min(254, len(s.Shape.Id))) > idSize {
			idSize = uint8(min(254, len(s.Shape.Id)))
		}
		if uint8(min(254, len(s.GetTripIdsString()))) > tIdsSize {
			tIdsSize = uint8(min(254, len(s.GetTripIdsString())))
		}
		if uint8(min(254, len(s.GetRouteIdsString()))) > rIdsSize {
			rIdsSize = uint8(min(254, len(s.GetRouteIdsString())))
		}
		if uint8(min(254, len(s.GetShortNamesString()))) > rShortNamesSize {
			rShortNamesSize = uint8(min(254, len(s.GetShortNamesString())))
		}
	}

	return []shp.Field{
		shp.StringField(sw.fldName("Id"), idSize),
		shp.StringField(sw.fldName("TripIds"), tIdsSize),
		shp.StringField(sw.fldName("RouteIds"), rIdsSize),
		shp.StringField(sw.fldName("RouteNames"), rShortNamesSize),
	}
}

/**
* Calculate the optimal shapefile attribute field sizes to hold aggregated trip/route fields
*/
func (sw *ShapeWriter) getFieldSizesForRouteShapes(shapes map[string]*AggrShape, typeMap map[int16]string, routeAddFlds []string, f *gtfsparser.Feed) []shp.Field {
	idSize := uint8(0)
	shortNameSize := uint8(0)
	LongNameSize := uint8(0)
	TypeNameSize := uint8(0)
	AgencyNameSize := uint8(0)
	AgencyUrlSize := uint8(0)

	addFldsSizes := make(map[string]uint8, len(routeAddFlds))

	for _, s := range shapes {
		for _, r := range s.Routes {
			if uint8(min(254, len(r.Id))) > idSize {
				idSize = uint8(min(254, len(r.Id)))
			}
			if uint8(min(254, len(r.Short_name))) > shortNameSize {
				shortNameSize = uint8(min(254, len(r.Short_name)))
			}
			if uint8(min(254, len(r.Long_name))) > LongNameSize {
				LongNameSize = uint8(min(254, len(r.Long_name)))
			}
			if str, ok := typeMap[r.Type]; ok {
				if uint8(min(254, len(str))) > TypeNameSize {
					TypeNameSize = uint8(min(254, len(str)))
				}
			} else {
				istr := strconv.FormatInt(int64(r.Type), 10)
				if uint8(min(254, len(istr))) > TypeNameSize {
					TypeNameSize = uint8(min(254, len(istr)))
				}
			}
			if uint8(min(254, len(r.Agency.Name))) > AgencyNameSize {
				AgencyNameSize = uint8(min(254, len(r.Agency.Name)))
			}
			if uint8(min(254, len(r.Agency.Url.String()))) > AgencyUrlSize {
				AgencyUrlSize = uint8(min(254, len(r.Agency.Url.String())))
			}

			for _, field := range routeAddFlds {
				if flds, ok := f.RoutesAddFlds[field]; ok {
					if val, ok := flds[r.Id]; ok {
						if uint8(min(254, len(val))) > addFldsSizes[field] {
							addFldsSizes[field] = uint8(min(254, len(val)))
						}
					}
				}
			}
		}
	}

	flds := []shp.Field{
		shp.StringField(sw.fldName("Route_id"), idSize),
		shp.StringField(sw.fldName("Short_name"), shortNameSize),
		shp.StringField(sw.fldName("Long_name"), LongNameSize),
		shp.StringField(sw.fldName("Type"), TypeNameSize),
		shp.NumberField(sw.fldName("Frequency"), 32),
		shp.FloatField(sw.fldName("Km_len"), 64, 10),
		shp.FloatField(sw.fldName("Km_tot"), 64, 10),
		shp.StringField(sw.fldName("Agency_name"), AgencyNameSize),
		shp.StringField(sw.fldName("Agency_url"), AgencyUrlSize),
		shp.FloatField(sw.fldName("Wchair_tr"), 32, 10),
		shp.FloatField(sw.fldName("Wchair_st"), 32, 10),
	}

	for _, field := range routeAddFlds {
		flds = append(flds, shp.StringField(sw.fldName(field), addFldsSizes[field]))
	}

	return flds
}

func (sw *ShapeWriter) fldName(f string) string {
	if n, ok := sw.fldMap[f]; ok {
		return n
	}
	return f
}

func (sw *ShapeWriter) getTripShapeFileName(in string) string {
	name := filepath.Base(in)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = fmt.Sprint(name, ".shape.shp")
	name = filepath.Join(filepath.Dir(in), name)
	return name
}

/**
* Return the sanitized output file name from the user-provided output file
*/
func (sw *ShapeWriter) getShapeFileName(in string) string {
	name := filepath.Base(in)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = fmt.Sprint(name, ".shp")
	name = filepath.Join(filepath.Dir(in), name)
	return name
}

/**
* Return the sanitized stations output file name from the user-provided output file
*/
func (sw *ShapeWriter) getShapeFileNameStations(in string) string {
	name := filepath.Base(in)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = fmt.Sprint(name, ".stations.shp")
	name = filepath.Join(filepath.Dir(in), name)
	return name
}

/**
* Return the sanitized trip stations output file name from the user-provided output file
*/
func (sw *ShapeWriter) getShapeFileNameTripStops(in string) string {
	name := filepath.Base(in)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = fmt.Sprint(name, ".stops.shp")
	name = filepath.Join(filepath.Dir(in), name)
	return name
}

/**
* Return the sanitized aggregate CSV output file name from the user-provided output file
*/
func (sw *ShapeWriter) getCsvFileName(in string) string {
	name := filepath.Base(in)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = fmt.Sprint(name, ".csv")
	name = filepath.Join(filepath.Dir(in), name)
	return name
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var DEG_TO_RAD float64 = 0.017453292519943295769236907684886127134428718885417254560
var DEG_TO_RAD32 float32 = float32(DEG_TO_RAD)

// Convert latitude/longitude to web mercator coordinates
func latLngToWebMerc(lat float32, lng float32) (float64, float64) {
	x := 6378137.0 * lng * DEG_TO_RAD32
	a := float64(lat * DEG_TO_RAD32)

	lng = x
	lat = float32(3189068.5 * math.Log((1.0+math.Sin(a))/(1.0-math.Sin(a))))
	return float64(lng), float64(lat)
}
