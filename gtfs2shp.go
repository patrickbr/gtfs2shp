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
	"os"
	"strconv"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gtfs2shp - 2016 by P. Brosi\n\nUsage:\n\n  %s -f <outputfile> -i <input GTFS>\n\nAllowed options:\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	gtfsPath := flag.String("i", "", "gtfs input path, zip or directory")
	shapeFilePath := flag.String("f", "out.shp", "shapefile output file")
	tripsExplicit := flag.Bool("t", false, "output each trip explicitly (creating a distinct geometry for every trip)")
	projection := flag.String("p", "4326", "output projection, either as SRID or as proj4 projection string")
	mots := flag.String("m", "0,1,2,3,4,5,6,7", "route types (MOT) to consider, as a comma separated list (see GTFS spec)")
	stations := flag.Bool("s", false, "output station point geometries as well (will be written into <outputfilename>-stations.shp)")

	flag.Parse()

	if len(*gtfsPath) == 0 {
		fmt.Fprintln(os.Stderr, "No GTFS location specified, see --help")
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error:", r)
		}
	}()

	sw := shape.NewShapeWriter(*projection, getMotMap(*mots))

	feed := gtfsparser.NewFeed()
	e := feed.Parse(*gtfsPath)

	if e != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing GTFS feed in '%s':\n ", *gtfsPath)
		fmt.Fprintf(os.Stderr, e.Error())
		os.Exit(1)
	} else {
		n := 0

		if *tripsExplicit {
			n += sw.WriteTripsExplicit(feed, *shapeFilePath)
		} else {
			n += sw.WriteShapes(feed, *shapeFilePath)
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

	ret := map[int16]bool{
		0: false,
		1: false,
		2: false,
		3: false,
		4: false,
		5: false,
		6: false,
		7: false,
	}

	for _, a := range arr {
		i, err := strconv.Atoi(a)
		if err == nil && i >= 0 && i < 8 {
			ret[int16(i)] = true
		}
	}

	return ret
}
