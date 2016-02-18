// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package main

import (
	"github.com/geops/gtfsparser"
	"github.com/patrickbr/gtfs2shp/shape"
	"fmt"
	"flag"
	"os"
)

func main() {
	flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "gtfs2shp - 2016 by P. Brosi\n\nUsage:\n\n  %s -f <outputfile> -i <input GTFS>\n\nArguments:\n\n", os.Args[0])
        flag.PrintDefaults()
	}

	gtfsPath := flag.String("i", "", "gtfs input path, zip or directory");
	shapeFilePath := flag.String("f", "out.shp", "shapefile output file");
	tripsExplicit := flag.Bool("t", false, "output each trip explicitly (creating a distinct geometry for every trip)");

	flag.Parse();


	if len(*gtfsPath) == 0 {
		fmt.Fprintln(os.Stderr, "No GTFS location specified, see --help");
		return;
	}

	feed := gtfsparser.NewFeed()
	e := feed.Parse(*gtfsPath)

	if e != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing GTFS feed in '%s':\n ", *gtfsPath);
		fmt.Fprintf(os.Stderr, e.Error())
	} else {		
		sw := shape.NewShapeWriter(4326);

		fmt.Printf("writing to %s\n", sw.GetShapeFileName(*shapeFilePath))

		if *tripsExplicit {
			sw.WriteTripsExplicit(feed, *shapeFilePath);
		} else {
			sw.WriteShapes(feed, *shapeFilePath);
		}
	}
}