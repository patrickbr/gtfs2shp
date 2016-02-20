// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package main

import (
	"flag"
	"fmt"
	"github.com/geops/gtfsparser"
	"github.com/patrickbr/gtfs2shp/shape"
	"os"
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

	sw := shape.NewShapeWriter(*projection)

	feed := gtfsparser.NewFeed()
	e := feed.Parse(*gtfsPath)

	if e != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing GTFS feed in '%s':\n ", *gtfsPath)
		fmt.Fprintf(os.Stderr, e.Error())
		os.Exit(1)
	} else {
		if *tripsExplicit {
			sw.WriteTripsExplicit(feed, *shapeFilePath)
		} else {
			sw.WriteShapes(feed, *shapeFilePath)
		}
	}
}
