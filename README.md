[![Go Report Card](https://goreportcard.com/badge/github.com/patrickbr/gtfs2shp)](https://goreportcard.com/report/github.com/patrickbr/gtfs2shp) [![Build Status](https://travis-ci.org/patrickbr/gtfs2shp.svg?branch=master)](https://travis-ci.org/patrickbr/gtfs2shp)

# gtfs2shp

Convert a [GTFS feed](https://developers.google.com/transit/gtfs/reference#routestxt) into an [ESRI shapefile](https://en.wikipedia.org/wiki/Shapefile).

## Credits

Creation of this tool was supported by [geOps, Freiburg](http://geops.de/).

## Installation

    $ go install github.com/patrickbr/gtfs2shp@latest

Requires golang version >= 1.7.

## Usage

Convert the [Chicago GTFS file](http://www.transitchicago.com/downloads/sch_data/) to a shapefile:

    $ gtfs2shp -i google_transit.zip -f output.shp

The primary entity will be the GTFS shapes. Routes/trips using this shapes will be stored in shapefile attributes as aggregated IDs and route short-names.

The result will look like this:

![Chicago Public Transit Network](https://patrickbrosi.de/chicago.png)

### Station geometries

If you also need the station geometries, just add the `-s` flag.

    $ gtfs2shp -i google_transit.zip -f output.shp -s

Station points along with all their GTFS attributes will be written into `<filename>.station.shp`, in the above case to `output.station.shp`.

### Explicit trips

If you need more trip/route information, use the `-t` mode. 

    $ gtfs2shp -i google_transit.zip -f output.shp -t
    
An explicit geometry together with all trip/route attributes will be written for each trip. Note that this will create redundant geometries.

### Coordinate reprojection

By default, coordinates will be outputted untouched as WGS84 (Lat/Lng) coordinates. If you need to reproject them, you can do so by using the `-p` parameter.

For example,

    $ gtfs2shp -i google_transit.zip -f output.shp -p 3857
    
will yield a shapefile with coordinates into Google Web Mercator (EPSG:3857) projection. Projections can either be specified as [EPSG codes](http://spatialreference.org/ref/epsg/) or as a [proj4 string](https://en.wikipedia.org/wiki/PROJ.4):

    $ gtfs2shp -i google_transit.zip -f output.shp -p "+proj=somerc +lat_0=46.95240555555556 +lon_0=7.439583333333333 +k_0=1 +x_0=600000 +y_0=200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs"

### MOT Filtering

By default, all vehicles defined in the GTFS feed will be included. You can specify which transportation types (MOTs) will be included in the output by setting the `-m` parameter to a comma separated list ot MOTs (as defined in the [GTFS ref](https://developers.google.com/transit/gtfs/reference#routes_route_type_field)). For example, to only output the rail network of Chicago, use:

    $ gtfs2shp -i google_transit.zip -f output.shp -m 1,2
    
## Flags
See

    $ gtfs2shp --help
    
for available command line arguments.

## License

See LICENSE.
