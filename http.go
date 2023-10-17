package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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
	LoadBalancer TraefikLoadBalancer `yaml:"loadBalancer,omitempty" json:"loadBalancer,omitempty"`
}

type TraefikLoadBalancer struct {
	Servers []string `yaml:"servers,omitempty" json:"servers,omitempty"`
}

func (hh *HTTPHandler) ProvideHTTPServices(w http.ResponseWriter, r *http.Request) {
	res := TraefikHTTPConfig{
		HTTP: TraefikHTTPServices{
			Services: make(map[string]TraefikService),
		},
	}

	for service, servers := range hh.im.ActiveServers.HTTP {
		res.HTTP.Services[service] = TraefikService{
			LoadBalancer: TraefikLoadBalancer{
				Servers: servers,
			},
		}
	}

	encoded, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprint("error marshaling json, err: ", err)))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(encoded)
}

func (hh *HTTPHandler) ProvideTCPServices(w http.ResponseWriter, r *http.Request) {
	res := TraefikTCPConfig{
		TCP: TraefikTCPServices{
			Services: make(map[string]TraefikService),
		},
	}

	for service, servers := range hh.im.ActiveServers.TCP {
		res.TCP.Services[service] = TraefikService{
			LoadBalancer: TraefikLoadBalancer{
				Servers: servers,
			},
		}
	}

	encoded, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprint("error marshaling json, err: ", err)))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(encoded)
}
