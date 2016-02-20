# gtfs2shp

Convert a [GTFS feed](https://developers.google.com/transit/gtfs/reference#routestxt) into an [ESRI shapefile](https://en.wikipedia.org/wiki/Shapefile).

## Credits

Creation of this tool was supported by [geOps, Freiburg](http://geops.de/).

## Installation

    $ go get github.com/patrickbr/gtfs2shp

## Usage

Convert the [Chicago GTFS file](http://www.transitchicago.com/downloads/sch_data/) to a shapefile:

    $ gtfs2shp -i google_transit.zip -f output.shp

The primary entity will be the GTFS shapes. Routes/trips using this shapes will be stored in shapefile attributes as aggregated IDs and route short-names.

The result will look like this:

![Alt text](http://patrickbrosi.de/chicago.png)

### Explicit trips

If you need more trip/route information, use the `-t` mode. 

    $ gtfs2shp -i google_transit.zip -f output.shp -t
    
An explicit geometry together with all trip/route attributes will be written for each trip. Not this will create redundant geometries.

### Coordinate reprojection

By default, coordinates will be outputted untouched as WGS84 (Lat/Lng) coordinates. If you need to reproject them, you can do so by using the `-p` parameter.

For example,

    $ gtfs2shp -i google_transit.zip -f output.shp -p 3857
    
will yield a shapefile with coordinates into Google Web Mercator (EPSG:3857) projection. Projections can either be specified as [EPSG codes](http://spatialreference.org/ref/epsg/) or as a [proj4 string](https://en.wikipedia.org/wiki/PROJ.4):

    $ gtfs2shp -i google_transit.zip -f output.shp -p "+proj=somerc +lat_0=46.95240555555556 +lon_0=7.439583333333333 +k_0=1 +x_0=600000 +y_0=200000 +ellps=bessel +towgs84=674.374,15.056,405.346,0,0,0,0 +units=m +no_defs"
    
## Flags
See

    $ gtfs2shp --help
    
for available command line arguments.

## License

See LICENSE.
