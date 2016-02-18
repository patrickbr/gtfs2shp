// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package shape

import (
	"github.com/geops/gtfsparser/gtfs"
	"strings"
)

type AggrShape struct {
	Shape *gtfs.Shape
	Trips map[string]*gtfs.Trip
	Routes map[string]*gtfs.Route
}

func NewAggrShape() *AggrShape {
	p := AggrShape{
		Trips:		make(map[string]*gtfs.Trip),
		Routes:		make(map[string]*gtfs.Route),
	};
	return &p
}

func (as *AggrShape) GetTripIdsString() string {
	keys := make([]string, 0, len(as.Trips))
    for k := range as.Trips {
        keys = append(keys, k)
    }

    return strings.Join(keys, ",");
}

func (as *AggrShape) GetRouteIdsString() string {
	keys := make([]string, 0, len(as.Routes))
    for k := range as.Routes {
        keys = append(keys, k)
    }

    return strings.Join(keys, ",");
}

func (as *AggrShape) GetShortNamesString() string {
	sNames := make([]string, 0, len(as.Routes))
    for _, v := range as.Routes {
        sNames = append(sNames, v.Short_name)
    }

    return strings.Join(sNames, ",");
}