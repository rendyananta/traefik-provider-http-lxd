package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"net/http"
)

const keyTraefikLBServers = "servers"

type HTTPHandler struct {
	im *InstanceManager
}

type TraefikHTTPConfig struct {
	HTTP TraefikHTTPServices `yaml:"http,omitempty" json:"http,omitempty"`
}

type TraefikTCPConfig struct {
	TCP TraefikTCPServices `yaml:"tcp,omitempty" json:"tcp,omitempty"`
}

type TraefikTCPServices struct {
	Services map[string]TraefikService `yaml:"services,omitempty" json:"services,omitempty"`
}

type TraefikHTTPServices struct {
	Services map[string]TraefikService `yaml:"services,omitempty" json:"services,omitempty"`
}

type TraefikService struct {
	LoadBalancer map[string]interface{} `yaml:"loadBalancer,omitempty" json:"loadBalancer,omitempty"`
}

func (hh *HTTPHandler) ProvideHTTPServices(w http.ResponseWriter, r *http.Request) {
	totalServices := len(hh.im.InstanceGroupRegistrar.Services)
	loadedServices := len(hh.im.ActiveServers.TCP) + len(hh.im.ActiveServers.HTTP)

	if totalServices == 0 {
		w.WriteHeader(http.StatusLocked)
		w.Write([]byte("loading config"))
		return
	}

	if loadedServices < totalServices && totalServices > 0 {
		w.WriteHeader(http.StatusLocked)
		w.Write([]byte(fmt.Sprintf("still fetching, loading: %d / %d", loadedServices, totalServices)))
		return
	}

	res := TraefikHTTPConfig{
		HTTP: TraefikHTTPServices{
			Services: make(map[string]TraefikService),
		},
	}

	for service, lb := range hh.im.ActiveServers.HTTP {
		lboptions := map[string]interface{}{}
		if lb.Options != nil {
			lboptions = lb.Options
		}

		ts := TraefikService{
			LoadBalancer: lboptions,
		}

		ts.LoadBalancer[keyTraefikLBServers] = lb.Servers
		res.HTTP.Services[service] = ts
	}

	encoded, err := yaml.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprint("error marshaling json, err: ", err)))
		return
	}

	w.Header().Add("content-type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(encoded)
}

func (hh *HTTPHandler) ProvideTCPServices(w http.ResponseWriter, r *http.Request) {
	totalServices := len(hh.im.InstanceGroupRegistrar.Services)
	loadedServices := len(hh.im.ActiveServers.TCP) + len(hh.im.ActiveServers.HTTP)

	if totalServices == 0 {
		w.WriteHeader(http.StatusLocked)
		w.Write([]byte("loading config"))
		return
	}

	if loadedServices < totalServices && totalServices > 0 {
		w.WriteHeader(http.StatusLocked)
		w.Write([]byte(fmt.Sprintf("still fetching, loading: %d / %d", loadedServices, totalServices)))
		return
	}

	res := TraefikTCPConfig{
		TCP: TraefikTCPServices{
			Services: make(map[string]TraefikService),
		},
	}

	for service, lb := range hh.im.ActiveServers.TCP {
		lboptions := map[string]interface{}{}
		if lb.Options != nil {
			lboptions = lb.Options
		}

		ts := TraefikService{
			LoadBalancer: lboptions,
		}

		ts.LoadBalancer[keyTraefikLBServers] = lb.Servers
		res.TCP.Services[service] = ts
	}

	encoded, err := yaml.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprint("error marshaling json, err: ", err)))
		return
	}

	w.Header().Add("content-type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(encoded)
}
