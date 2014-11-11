package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/docker/libcluster"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

type HttpApiFunc func(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request)

func getContainersJSON(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	all := r.Form.Get("all") == "1"

	out := []dockerclient.Container{}
	for _, container := range c.Containers() {
		// Skip stopped containers unless -a was specified.
		if !strings.Contains(container.Status, "Up") && !all {
			continue
		}
		out = append(out, container.Container)
	}

	sort.Sort(sort.Reverse(ContainerSorter(out)))
	json.NewEncoder(w).Encode(out)
}

func ping(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

func notImplementedHandler(c *libcluster.Cluster, w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not supported in clustering mode.", http.StatusNotImplemented)
}

func createRouter(c *libcluster.Cluster) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":                          ping,
			"/events":                         notImplementedHandler,
			"/info":                           notImplementedHandler,
			"/version":                        notImplementedHandler,
			"/images/json":                    notImplementedHandler,
			"/images/viz":                     notImplementedHandler,
			"/images/search":                  notImplementedHandler,
			"/images/get":                     notImplementedHandler,
			"/images/{name:.*}/get":           notImplementedHandler,
			"/images/{name:.*}/history":       notImplementedHandler,
			"/images/{name:.*}/json":          notImplementedHandler,
			"/containers/ps":                  getContainersJSON,
			"/containers/json":                getContainersJSON,
			"/containers/{name:.*}/export":    notImplementedHandler,
			"/containers/{name:.*}/changes":   notImplementedHandler,
			"/containers/{name:.*}/json":      notImplementedHandler,
			"/containers/{name:.*}/top":       notImplementedHandler,
			"/containers/{name:.*}/logs":      notImplementedHandler,
			"/containers/{name:.*}/attach/ws": notImplementedHandler,
		},
		"POST": {
			"/auth":                         notImplementedHandler,
			"/commit":                       notImplementedHandler,
			"/build":                        notImplementedHandler,
			"/images/create":                notImplementedHandler,
			"/images/load":                  notImplementedHandler,
			"/images/{name:.*}/push":        notImplementedHandler,
			"/images/{name:.*}/tag":         notImplementedHandler,
			"/containers/create":            notImplementedHandler,
			"/containers/{name:.*}/kill":    notImplementedHandler,
			"/containers/{name:.*}/pause":   notImplementedHandler,
			"/containers/{name:.*}/unpause": notImplementedHandler,
			"/containers/{name:.*}/restart": notImplementedHandler,
			"/containers/{name:.*}/start":   notImplementedHandler,
			"/containers/{name:.*}/stop":    notImplementedHandler,
			"/containers/{name:.*}/wait":    notImplementedHandler,
			"/containers/{name:.*}/resize":  notImplementedHandler,
			"/containers/{name:.*}/attach":  notImplementedHandler,
			"/containers/{name:.*}/copy":    notImplementedHandler,
			"/containers/{name:.*}/exec":    notImplementedHandler,
			"/exec/{name:.*}/start":         notImplementedHandler,
			"/exec/{name:.*}/resize":        notImplementedHandler,
		},
		"DELETE": {
			"/containers/{name:.*}": notImplementedHandler,
			"/images/{name:.*}":     notImplementedHandler,
		},
		"OPTIONS": {
			"": notImplementedHandler,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			log.Printf("Registering %s, %s", method, route)
			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("-> %s %s\n", r.Method, r.RequestURI)
				localFct(c, w, r)
			}
			localMethod := method

			// add the new route
			r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return r, nil
}

func ListenAndServe(c *libcluster.Cluster, addr string) error {
	r, err := createRouter(c)
	if err != nil {
		return err
	}
	s := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s.ListenAndServe()
}
