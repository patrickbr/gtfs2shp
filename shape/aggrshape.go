// Copyright 2016 Patrick Brosi
// Authors: info@patrickbrosi.de
//
// Use of this source code is governed by a GPL v2
// license that can be found in the LICENSE file

package shape

import (
	"github.com/patrickbr/gtfsparser/gtfs"
	"strings"
)

// AggrShape is a trip-aggregated shapes containing
// gtfs.Route and gtfs.Trip objects sharing the
// same shape
type AggrShape struct {
	Shape  *gtfs.Shape
	Trips  map[string]*gtfs.Trip
	Routes map[string]*gtfs.Route
}

// NewAggrShape returns a new AggrShape instance
func NewAggrShape() *AggrShape {
	p := AggrShape{
		Trips:  make(map[string]*gtfs.Trip),
		Routes: make(map[string]*gtfs.Route),
	}
	return &p
}

// GetTripIdsString returns a comma separated list of
// trip IDs contained in this AggrShape
func (as *AggrShape) GetTripIdsString() string {
	keys := make([]string, 0, len(as.Trips))
	for k := range as.Trips {
		keys = append(keys, k)
	}

	return strings.Join(keys, ",")
}

// GetRouteIdsString returns a comma separated list of
// route IDs contained in this AggrShape
func (as *AggrShape) GetRouteIdsString() string {
	keys := make(map[string]struct{})
	for k := range as.Routes {
		keys[k] = struct{}{}
	}

	ids := make([]string, 0)
	for k := range keys {
		ids = append(ids, k)
	}

	return strings.Join(ids, ",")
}

// GetShortNamesString returns a comma separated list of
// the short names of the routes contained in this AggrShape
func (as *AggrShape) GetShortNamesString() string {
	sNames := make(map[string]struct{})
	for _, v := range as.Routes {
		sNames[v.Short_name] = struct{}{}
	}

	sNamesSl := make([]string, 0)
	for k := range sNames {
		sNamesSl = append(sNamesSl, k)
	}

	return strings.Join(sNamesSl, ",")
}
