// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package shape

import (
	"github.com/geops/gtfsparser"
	"github.com/geops/gtfsparser/gtfs"
	"github.com/jonas-p/go-shp"
	"fmt"
	"path/filepath"
	"strings"
)

type ShapeWriter struct {
	srid int
}

func NewShapeWriter(srid int) *ShapeWriter {
	p := ShapeWriter{srid}
	return &p
}

func (sw *ShapeWriter) WriteTripsExplicit(f *gtfsparser.Feed, outFile string) int {
	shape, err := shp.Create(sw.GetShapeFileName(outFile), shp.POLYLINE)

	if err != nil {fmt.Println(err)}
	defer shape.Close()

	shape.SetFields(sw.GetFieldSizesForTrips(f.Trips));

	n := 0;
	calcedShapes := make(map[string]*shp.PolyLine);

	// iterate through trips
	for _, trip := range f.Trips {
		if trip.Shape != nil {
			points := sw.GtfsShapePointsToShpLinePoints(trip.Shape.Points);
			parts := [][]shp.Point{points};

			// prevent re-calcing of polylines for each trips
			if val, ok := calcedShapes[trip.Shape.Id]; ok {
    			shape.Write(val);
    		} else {
	    		calcedShapes[trip.Shape.Id] = shp.NewPolyLine(parts);
				shape.Write(calcedShapes[trip.Shape.Id]);
			}
		} else {
			// use station positions as polyline anchors
			points := sw.GtfsStationPointsToShpLinePoints(trip.StopTimes);
			parts := [][]shp.Point{points};
			
			shape.Write(shp.NewPolyLine(parts));
		}

		shape.WriteAttribute(n, 0, trip.Id);
		shape.WriteAttribute(n, 1, trip.Headsign);
		shape.WriteAttribute(n, 2, trip.Short_name);
		shape.WriteAttribute(n, 3, trip.Direction_id);
		shape.WriteAttribute(n, 4, trip.Block_id);
		shape.WriteAttribute(n, 5, trip.Wheelchair_accessible);
		shape.WriteAttribute(n, 6, trip.Bikes_allowed);
		shape.WriteAttribute(n, 7, trip.Route.Short_name);
		shape.WriteAttribute(n, 8, trip.Route.Long_name);
		shape.WriteAttribute(n, 9, trip.Route.Desc);
		shape.WriteAttribute(n, 10, trip.Route.Type);
		shape.WriteAttribute(n, 11, trip.Route.Url);
		shape.WriteAttribute(n, 12, trip.Route.Color);
		shape.WriteAttribute(n, 13, trip.Route.Text_color);

		n = n+1;
	}

	return n
}

func (sw *ShapeWriter) WriteShapes(f *gtfsparser.Feed, outFile string) int {
	shape, err := shp.Create(sw.GetShapeFileName(outFile), shp.POLYLINE)

	if err != nil {fmt.Println(err)}
	defer shape.Close()

	n := 0;

	// get aggreshape map
	aggrShapes := sw.getAggrShapes(f.Trips);
	shape.SetFields(sw.GetFieldSizesForShapes(aggrShapes));
	
	for _, aggrShape := range aggrShapes {
		points := sw.GtfsShapePointsToShpLinePoints(aggrShape.Shape.Points);
		parts := [][]shp.Point{points};

		shape.Write(shp.NewPolyLine(parts));

		shape.WriteAttribute(n, 0, aggrShape.Shape.Id);
		shape.WriteAttribute(n, 1, aggrShape.GetTripIdsString());
		shape.WriteAttribute(n, 2, aggrShape.GetRouteIdsString());
		shape.WriteAttribute(n, 3, aggrShape.GetShortNamesString());

		n = n+1;
	}

	return n
}

func (sw *ShapeWriter) getAggrShapes(trips map[string]*gtfs.Trip) map[string]*AggrShape {
	ret := make(map[string]*AggrShape);

	// iterate through all trips
	for _, trip := range trips {
		if trip.Shape == nil { continue; }

		// check if shape is already present
		if _, ok := ret[trip.Shape.Id]; !ok {
			ret[trip.Shape.Id] = NewAggrShape();
			ret[trip.Shape.Id].Shape = trip.Shape;
    	}

	    ret[trip.Shape.Id].Trips[trip.Id] = trip;
		ret[trip.Shape.Id].Routes[trip.Route.Id] = trip.Route
	}


	return ret;
}

func (sw *ShapeWriter) GtfsShapePointsToShpLinePoints(gtfsshape gtfs.ShapePoints) []shp.Point {
	ret := make([]shp.Point, len(gtfsshape));
	for i, pt := range gtfsshape {
	  ret[i].Y = float64(pt.Lat);
	  ret[i].X = float64(pt.Lon);
	}

	return ret;
}

func (sw *ShapeWriter) GtfsStationPointsToShpLinePoints(stoptimes gtfs.StopTimes) []shp.Point {
	ret := make([]shp.Point, len(stoptimes));
	for i, st := range stoptimes {
	  ret[i].Y = float64(st.Stop.Lat);
	  ret[i].X = float64(st.Stop.Lon);
	}

	return ret;
}

func (sw *ShapeWriter) GetFieldSizesForTrips(trips map[string]*gtfs.Trip) []shp.Field {
	idSize := uint8(0);
	headsignSize := uint8(0);
	shortNameSize := uint8(0);
	blockIdSize := uint8(0);
	rShortNameSize := uint8(0);
	rLongNameSize := uint8(0);
	rDescSize := uint8(0);
	rUrlSize := uint8(0);
	rColorSize := uint8(0);
	rTextColorSize := uint8(0);

	for _, st := range trips {
		if uint8(min(254, len(st.Id))) > idSize { idSize = uint8(min(254, len(st.Id))) }
		if uint8(min(254, len(st.Headsign))) > headsignSize { headsignSize = uint8(min(254, len(st.Headsign))) }
		if uint8(min(254, len(st.Short_name))) > shortNameSize { shortNameSize = uint8(min(254, len(st.Short_name))) }
		if uint8(min(254, len(st.Block_id))) > blockIdSize { blockIdSize = uint8(min(254, len(st.Block_id))) }
		if uint8(min(254, len(st.Route.Short_name))) > rShortNameSize { rShortNameSize = uint8(min(254, len(st.Route.Short_name))) }
		if uint8(min(254, len(st.Route.Long_name))) > rLongNameSize { rLongNameSize = uint8(min(254, len(st.Route.Long_name))) }
		if uint8(min(254, len(st.Route.Desc))) > rDescSize { rDescSize = uint8(min(254, len(st.Route.Desc))) }
		if uint8(min(254, len(st.Route.Url))) > rUrlSize { rUrlSize = uint8(min(254, len(st.Route.Url))) }
		if uint8(min(254, len(st.Route.Color))) > rColorSize { rColorSize = uint8(min(254, len(st.Route.Color))) }
		if uint8(min(254, len(st.Route.Text_color))) > rTextColorSize { rTextColorSize = uint8(min(254, len(st.Route.Text_color))) }
	}

	return []shp.Field{
	    shp.StringField("Id", idSize),
	    shp.StringField("Headsign", headsignSize),
	    shp.StringField("ShortName", shortNameSize),
	    shp.NumberField("Direction_id", 10),
	    shp.StringField("BlockId", blockIdSize),
	    shp.NumberField("Wheelchair_accessible", 2),
	    shp.NumberField("Bikes_allowed", 2),
	    shp.StringField("R_ShortName", rShortNameSize),
	    shp.StringField("R_LongName", rLongNameSize),
	    shp.StringField("R_Desc", rDescSize),
	    shp.NumberField("R_Type", 10),
	    shp.StringField("R_URL", rUrlSize),
	    shp.StringField("R_Color", rColorSize),
	    shp.StringField("R_TextColor", rTextColorSize),
	}
}


func (sw *ShapeWriter) GetFieldSizesForShapes(shapes map[string]*AggrShape) []shp.Field {
	idSize := uint8(0);
	tIdsSize := uint8(0);
	rIdsSize := uint8(0);
	rShortNamesSize := uint8(0);

	for _, s := range shapes {
		if uint8(min(254, len(s.Shape.Id))) > idSize { idSize = uint8(min(254, len(s.Shape.Id))) }
		if uint8(min(254, len(s.GetTripIdsString()))) > tIdsSize { tIdsSize = uint8(min(254, len(s.GetTripIdsString()))) }
		if uint8(min(254, len(s.GetRouteIdsString()))) > rIdsSize { rIdsSize = uint8(min(254, len(s.GetRouteIdsString()))) }
		if uint8(min(254, len(s.GetShortNamesString()))) > rShortNamesSize { rShortNamesSize = uint8(min(254, len(s.GetShortNamesString()))) }
	}

	return []shp.Field{
	    shp.StringField("Id", idSize),
	    shp.StringField("TripIds", tIdsSize),
	    shp.StringField("RouteIds", rIdsSize),
	    shp.StringField("RouteSNames", rShortNamesSize),
	}
}

func (sw *ShapeWriter) GetShapeFileName(in string) string {
	name := filepath.Base(in);
	name = strings.TrimSuffix(name, filepath.Ext(name));
	name = fmt.Sprint(name, ".shp");
	name = filepath.Join(filepath.Dir(in), name);
	return name;
}

func min(a, b int) int {
   if a < b {
      return a
   }
   return b
}