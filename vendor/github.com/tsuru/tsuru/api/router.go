// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/tsuru/tsuru/api/context"
)

type delayedRouter struct {
	mux.Router
}

func (r *delayedRouter) registerVars(req *http.Request, vars map[string]string) {
	values := make(url.Values)
	for key, value := range vars {
		values[":"+key] = []string{value}
	}
	req.URL.RawQuery = url.Values(values).Encode() + "&" + req.URL.RawQuery
}

func (r *delayedRouter) Add(method string, path string, h http.Handler) *mux.Route {
	return r.Router.Handle(path, h).Methods(method)
}

// AddAll binds a path to GET, POST, PUT and DELETE methods.
func (r *delayedRouter) AddAll(path string, h http.Handler) *mux.Route {
	return r.Router.Handle(path, h).Methods("GET", "POST", "PUT", "DELETE")
}

func (r *delayedRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var match mux.RouteMatch
	if !r.Match(req, &match) {
		http.NotFound(w, req)
		return
	}
	r.registerVars(req, match.Vars)
	context.SetDelayedHandler(req, match.Handler)
}
