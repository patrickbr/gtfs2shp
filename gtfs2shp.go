// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package main

import (
	"flag"
	"fmt"
	"github.com/patrickbr/gtfs2shp/shape"
	"github.com/patrickbr/gtfsparser"
	gtfs "github.com/patrickbr/gtfsparser/gtfs"
	"os"
	"strconv"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gtfs2shp - 2016 by P. Brosi\n\nUsage:\n\n  %s -f <outputfile> -i <input GTFS>\n\nAllowed options:\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	routeTypeMapping := make(map[int16]string, 0)
	routeAddFlds := make(map[string]bool, 0)

	gtfsPath := flag.String("i", "", "gtfs input path, zip or directory")
	shapeFilePath := flag.String("f", "out.shp", "shapefile output file")
	tripsExplicit := flag.Bool("t", false, "output each trip explicitly (creating a distinct geometry for every trip)")
	perRoute := flag.Bool("r", false, "output shapes per route")
	projection := flag.String("p", "4326", "output projection, either as SRID or as proj4 projection string")
	mots := flag.String("m", "", "route types (MOT) to consider, as a comma separated list (see GTFS spec). Empty keeps all.")
	stations := flag.Bool("s", false, "output station point geometries as well (will be written into <outputfilename>-stations.shp)")
	routeTypeNameMapping := flag.String("route-type-mapping", "", "semicolon-separated list of mapping of {route_type}:{string} to be used on output")
	writeAddRouteFlds := flag.String("write-add-route-fields", "", "semicolon-separated list of additional route fields to be included in output")
	writeRouteOverviewCsv := flag.Bool("write-route-overview-csv", false, "write a route overview CSV")

	flag.Parse()

	if len(*gtfsPath) == 0 {
		fmt.Fprintln(os.Stderr, "No GTFS location specified, see --help")
		os.Exit(1)
	}

	for _, pairs := range strings.Split(*routeTypeNameMapping, ";") {
		if len(pairs) == 0 {
			continue
		}
		tupl := strings.SplitN(pairs, ":", 2)

		if len(tupl) != 2 {
			fmt.Println("Could not read mapping tuple", pairs)
			os.Exit(1)
		}

		mot, e := strconv.Atoi(tupl[0])

		if e != nil {
			fmt.Println(e)
			os.Exit(1)
		}

		routeTypeMapping[int16(mot)] = tupl[1]
	}

	for _, field := range strings.Split(*writeAddRouteFlds, ";") {
		if len(field) == 0 {
			continue
		}

		routeAddFlds[field] = true
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error:", r)
		}
	}()

	sw := shape.NewShapeWriter(*projection, getMotMap(*mots))

	feed := gtfsparser.NewFeed()
	feed.SetParseOpts(gtfsparser.ParseOptions{false, false, false, false, "", false, false, false, len(routeAddFlds) > 0, gtfs.Date{}, gtfs.Date{}, make([]gtfsparser.Polygon, 0), false, make(map[int16]bool, 0), make(map[int16]bool, 0), false})
	e := feed.Parse(*gtfsPath)

	if e != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing GTFS feed in '%s':\n ", *gtfsPath)
		fmt.Fprintf(os.Stderr, e.Error())
		os.Exit(1)
	} else {
		n := 0

		if *tripsExplicit {
			n += sw.WriteTripsExplicit(feed, *shapeFilePath)
		} else if *perRoute {
			n += sw.WriteRouteShapes(feed, routeTypeMapping, routeAddFlds, *shapeFilePath)
		} else {
			n += sw.WriteShapes(feed, *shapeFilePath)
		}

		if *writeRouteOverviewCsv {
			sw.WriteRouteOverviewCsv(feed, routeTypeMapping, routeAddFlds, *shapeFilePath)
		}

		// write stations if requested
		if *stations {
			n += sw.WriteStops(feed, *shapeFilePath)
		}

		fmt.Printf("Written %d geometries.\n", n)
	}
}

func getMotMap(motList string) map[int16]bool {
	arr := strings.Split(motList, ",")

	ret := map[int16]bool{}

	for _, a := range arr {
		i, err := strconv.Atoi(a)
		if err == nil && i >= 0 && i < 8 {
			ret[int16(i)] = true
		}
	}

	return ret
}
