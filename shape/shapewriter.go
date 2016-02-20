// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package shape

import (
	"fmt"
	"github.com/geops/gtfsparser"
	"github.com/geops/gtfsparser/gtfs"
	"github.com/jonas-p/go-shp"
	"github.com/pebbe/go-proj-4/proj"
	"path/filepath"
	"strconv"
	"strings"
)

var WGS84_STR = "+proj=longlat +ellps=WGS84 +datum=WGS84 +no_defs"

type ShapeWriter struct {
	outProj   *proj.Proj
	wgs84Proj *proj.Proj
}

/**
 * Create a new ShapeWriter, writing in the specified projection (as proj4 string)
 */
func NewShapeWriter(projection string) *ShapeWriter {
	sw := ShapeWriter{}

	/**
	 * NOTE: go-proj-4 does not yet support pj_is_latlong(), which
	 * means we have no secure way the test whether the user requested
	 * latlng output. If EPSG:4326 is defined in another way than tested
	 * here, it will reproject the coordinates to latlng in radians!
	 */
	if projection != "4326" && projection != WGS84_STR {
		// we need reprojection of coordinates

		var targetProj *proj.Proj
		wgs84, err := proj.NewProj(WGS84_STR)
		if err != nil {
			panic(fmt.Sprintf("Could not init WGS84 projection, maybe proj4 is not available? (%s)", projection, err))
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

/**
 * Write the shapes contained in Feed f to outFile, with each trip as an
 * explicit geometry with all trip attributes
 */
func (sw *ShapeWriter) WriteTripsExplicit(f *gtfsparser.Feed, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileName(outFile), shp.POLYLINE)

	if err != nil {
		fmt.Println(err)
	}
	defer shape.Close()

	shape.SetFields(sw.getFieldSizesForTrips(f.Trips))

	n := 0
	calcedShapes := make(map[string]*shp.PolyLine)

	// iterate through trips
	for _, trip := range f.Trips {
		if trip.Shape != nil {
			points := sw.gtfsShapePointsToShpLinePoints(trip.Shape.Points)
			parts := [][]shp.Point{points}

			// prevent re-calcing of polylines for each trips
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
		shape.WriteAttribute(n, 1, trip.Headsign)
		shape.WriteAttribute(n, 2, trip.Short_name)
		shape.WriteAttribute(n, 3, trip.Direction_id)
		shape.WriteAttribute(n, 4, trip.Block_id)
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

	return n
}

/**
 * Write the shapes contained in Feed f to outFile, with each shape containing
 * aggregrated trip/route information
 */
func (sw *ShapeWriter) WriteShapes(f *gtfsparser.Feed, outFile string) int {
	shape, err := shp.Create(sw.getShapeFileName(outFile), shp.POLYLINE)

	if err != nil {
		fmt.Println(err)
	}
	defer shape.Close()

	n := 0

	// get aggreshape map
	aggrShapes := sw.getAggrShapes(f.Trips)
	shape.SetFields(sw.getFieldSizesForShapes(aggrShapes))

	for _, aggrShape := range aggrShapes {
		points := sw.gtfsShapePointsToShpLinePoints(aggrShape.Shape.Points)
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

/**
 * Return aggregrated shapes from GTFS trips
 */
func (sw *ShapeWriter) getAggrShapes(trips map[string]*gtfs.Trip) map[string]*AggrShape {
	ret := make(map[string]*AggrShape)

	// iterate through all trips
	for _, trip := range trips {
		if trip.Shape == nil {
			continue
		}

		// check if shape is already present
		if _, ok := ret[trip.Shape.Id]; !ok {
			ret[trip.Shape.Id] = NewAggrShape()
			ret[trip.Shape.Id].Shape = trip.Shape
		}

		ret[trip.Shape.Id].Trips[trip.Id] = trip
		ret[trip.Shape.Id].Routes[trip.Route.Id] = trip.Route
	}

	return ret
}

/**
 * Returns a shapefile geometry from a GTFS shape, reprojected
 */
func (sw *ShapeWriter) gtfsShapePointsToShpLinePoints(gtfsshape gtfs.ShapePoints) []shp.Point {
	ret := make([]shp.Point, len(gtfsshape))
	for i, pt := range gtfsshape {
		if sw.outProj != nil {
			x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(pt.Lon)), proj.DegToRad(float64(pt.Lat)))
			ret[i].Y = y
			ret[i].X = x
		} else {
			ret[i].Y = float64(pt.Lat)
			ret[i].X = float64(pt.Lon)
		}
	}

	return ret
}

/**
 * Returns a shapefile geometry from a GTFS station list (if shapes are not available in the feed), reprojected
 */
func (sw *ShapeWriter) gtfsStationPointsToShpLinePoints(stoptimes gtfs.StopTimes) []shp.Point {
	ret := make([]shp.Point, len(stoptimes))
	for i, st := range stoptimes {
		if sw.outProj != nil {
			x, y, _ := proj.Transform2(sw.wgs84Proj, sw.outProj, proj.DegToRad(float64(st.Stop.Lon)), proj.DegToRad(float64(st.Stop.Lat)))
			ret[i].Y = y
			ret[i].X = x
		} else {
			ret[i].Y = float64(st.Stop.Lat)
			ret[i].X = float64(st.Stop.Lon)
		}
	}

	return ret
}

/**
 * Calculate the optimal shapefile attribute field sizes to hold trip/route fields
 */
func (sw *ShapeWriter) getFieldSizesForTrips(trips map[string]*gtfs.Trip) []shp.Field {
	idSize := uint8(0)
	headsignSize := uint8(0)
	shortNameSize := uint8(0)
	blockIdSize := uint8(0)
	rShortNameSize := uint8(0)
	rLongNameSize := uint8(0)
	rDescSize := uint8(0)
	rUrlSize := uint8(0)
	rColorSize := uint8(0)
	rTextColorSize := uint8(0)

	for _, st := range trips {
		if uint8(min(254, len(st.Id))) > idSize {
			idSize = uint8(min(254, len(st.Id)))
		}
		if uint8(min(254, len(st.Headsign))) > headsignSize {
			headsignSize = uint8(min(254, len(st.Headsign)))
		}
		if uint8(min(254, len(st.Short_name))) > shortNameSize {
			shortNameSize = uint8(min(254, len(st.Short_name)))
		}
		if uint8(min(254, len(st.Block_id))) > blockIdSize {
			blockIdSize = uint8(min(254, len(st.Block_id)))
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
		if uint8(min(254, len(st.Route.Url))) > rUrlSize {
			rUrlSize = uint8(min(254, len(st.Route.Url)))
		}
		if uint8(min(254, len(st.Route.Color))) > rColorSize {
			rColorSize = uint8(min(254, len(st.Route.Color)))
		}
		if uint8(min(254, len(st.Route.Text_color))) > rTextColorSize {
			rTextColorSize = uint8(min(254, len(st.Route.Text_color)))
		}
	}

	return []shp.Field{
		shp.StringField("Id", idSize),
		shp.StringField("Headsign", headsignSize),
		shp.StringField("ShortName", shortNameSize),
		shp.NumberField("Dir_id", 1),
		shp.StringField("BlockId", blockIdSize),
		shp.NumberField("Wheelchr_a", 1),
		shp.NumberField("Bikes_alwd", 1),
		shp.StringField("R_ShrtName", rShortNameSize),
		shp.StringField("R_LongName", rLongNameSize),
		shp.StringField("R_Desc", rDescSize),
		shp.NumberField("R_Type", 1),
		shp.StringField("R_URL", rUrlSize),
		shp.StringField("R_Color", rColorSize),
		shp.StringField("R_TextColor", rTextColorSize),
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
		shp.StringField("Id", idSize),
		shp.StringField("TripIds", tIdsSize),
		shp.StringField("RouteIds", rIdsSize),
		shp.StringField("RouteNames", rShortNamesSize),
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
