// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go-bindata --nocompress --nometadata --pkg ctrlz -o assets.gen.go assets/...

// Package ctrlz implements Istio's introspection facility. When components
// integrate with ControlZ, they automatically gain an IP port which allows operators
// to visualize and control a number of aspects of each process, including controlling
// logging scopes, viewing command-line options, memory use, etc. Additionally,
// the port implements a REST API allowing access and control over the same state.
//
// ControlZ is designed around the idea of "topics". A topic corresponds to the different
// parts of the UI. There are a set of built-in topics representing the core introspection
// functionality, and each component that uses ControlZ can add new topics specialized
// for their purpose.
package ctrlz

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"sync"

	"istio.io/istio/pkg/ctrlz/fw"
	"istio.io/istio/pkg/ctrlz/topics"
	"istio.io/istio/pkg/log"
)

var coreTopics = []fw.Topic{
	topics.ScopeTopic(),
	topics.MemTopic(),
	topics.EnvTopic(),
	topics.ProcTopic(),
	topics.ArgsTopic(),
	topics.VersionTopic(),
	topics.MetricsTopic(),
}

var allTopics []fw.Topic
var topicMutex sync.Mutex
var server *http.Server
var shutdown sync.WaitGroup
var shutdownMutex sync.Mutex
var abort = false

func augmentLayout(layout *template.Template, page string) *template.Template {
	return template.Must(layout.Parse(string(MustAsset(page))))
}

func registerTopic(router *mux.Router, layout *template.Template, t fw.Topic) {
	htmlRouter := router.NewRoute().PathPrefix("/" + t.Prefix() + "z").Subrouter()
	jsonRouter := router.NewRoute().PathPrefix("/" + t.Prefix() + "j").Subrouter()

	tmpl := template.Must(template.Must(layout.Clone()).Parse("{{ define \"title\" }}" + t.Title() + "{{ end }}"))
	t.Activate(fw.NewContext(htmlRouter, jsonRouter, tmpl))
}

// getLocalIP returns a non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback then return it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

type topic struct {
	Name string
	URL  string
}

func getTopics() []topic {
	var topics []topic

	topicMutex.Lock()
	defer topicMutex.Unlock()

	for _, t := range allTopics {
		topics = append(topics, topic{Name: t.Title(), URL: "/" + t.Prefix() + "z/"})
	}

	return topics
}

// RegisterTopic registers a new Control-Z topic for the current process.
func RegisterTopic(t fw.Topic) {
	topicMutex.Lock()
	defer topicMutex.Unlock()

	allTopics = append(allTopics, t)
}

// Run starts up the ControlZ listeners.
//
// ControlZ uses the set of standard core topics, the
// supplied custom topics, as well as any topics registered
// via the RegisterTopic function.
func Run(o *Options, customTopics []fw.Topic) {
	if o.Port == 0 {
		// disabled
		return
	}

	topicMutex.Lock()

	for _, t := range coreTopics {
		allTopics = append(allTopics, t)
	}

	for _, t := range customTopics {
		allTopics = append(allTopics, t)
	}

	topicMutex.Unlock()

	exec, _ := os.Executable()
	instance := exec + " - " + getLocalIP()

	funcs := template.FuncMap{
		"getTopics": getTopics,
	}

	baseLayout := template.Must(template.New("base").Parse(string(MustAsset("assets/templates/layouts/base.html"))))
	baseLayout = baseLayout.Funcs(funcs)
	baseLayout = template.Must(baseLayout.Parse("{{ define \"instance\" }}" + instance + "{{ end }}"))
	_ = augmentLayout(baseLayout, "assets/templates/modules/header.html")
	_ = augmentLayout(baseLayout, "assets/templates/modules/sidebar.html")
	_ = augmentLayout(baseLayout, "assets/templates/modules/last-refresh.html")
	mainLayout := augmentLayout(template.Must(baseLayout.Clone()), "assets/templates/layouts/main.html")

	router := mux.NewRouter()
	for _, t := range allTopics {
		registerTopic(router, mainLayout, t)
	}

	registerHome(router, mainLayout)

	addr := o.Address
	if addr == "*" {
		addr = ""
	}

	shutdownMutex.Lock()

	if abort {
		// reset to default state in case Run is called again
		abort = false
	} else {
		server = &http.Server{
			Addr:           fmt.Sprintf("%s:%d", addr, o.Port),
			Handler:        router,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		shutdown.Add(1)
	}

	shutdownMutex.Unlock()

	if server != nil {
		log.Infof("ControlZ available at %s:%d", getLocalIP(), o.Port)
		server.ListenAndServe()
		log.Infof("ControlZ terminated")
		shutdown.Done()
		server = nil
	}
}

// Stop terminates ControlZ.
//
// Stop is not normally used by programs that expose ControlZ, it is primarily intended to be
// used by tests.
func Stop() {
	wait := false

	shutdownMutex.Lock()
	if server != nil {
		server.Close()
		wait = true
	} else {
		// prevent any in-progress calls to Run from succeeding
		abort = true
	}
	shutdownMutex.Unlock()

	if wait {
		shutdown.Wait()
	}
}
