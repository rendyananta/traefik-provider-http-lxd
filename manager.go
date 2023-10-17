package main

import (
	"bytes"
	"fmt"
	"github.com/canonical/lxd/shared/api"
	"gopkg.in/yaml.v3"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"
	"traefik-http-lxd-provider/client"
)

type VirtualInstanceGroup struct {
	ServiceType    ServiceType            `yaml:"service_type"`
	GroupName      GroupName              `yaml:"group_name"`
	ProjectName    ProjectName            `yaml:"project_name"`
	InstancePrefix string                 `yaml:"instance_prefix"`
	InstanceType   api.InstanceType       `yaml:"instance_type"`
	Port           int                    `yaml:"port"`
	LBOptions      map[string]interface{} `yaml:"lb_options,omitempty"`
	Members        []InstanceInfo
}

type InstanceInfo struct {
	Name        string
	ProfileName string
	V4Addr      string
	V6Addr      string
}

type ProjectName = string
type GroupName = string

type ServiceType = string

const ServiceTypeHTTP ServiceType = "http"
const ServiceTypeTCP ServiceType = "tcp"

type GoroutineWorker interface {
	AddTask(task func())
}

type LXDClientPool interface {
	Get() (*client.LXDClient, error)
	Release(conn *client.LXDClient) error
}

type VirtualInstanceGroupRegistrar struct {
	Services map[string]VirtualInstanceGroup `yaml:"services"`
}

type LoadBalancerServers struct {
	Servers []string
	Options map[string]interface{}
}

type ActiveServers struct {
	HTTP map[string]LoadBalancerServers
	TCP  map[string]LoadBalancerServers
}

type InstanceManager struct {
	clientPool             LXDClientPool
	worker                 GoroutineWorker
	InstanceGroupRegistrar VirtualInstanceGroupRegistrar
	ActiveServers          ActiveServers
}

const defaultHTTPPort = 80

func NewInstanceManager(worker GoroutineWorker, pool LXDClientPool) (*InstanceManager, error) {
	c := &InstanceManager{
		worker:     worker,
		clientPool: pool,
		ActiveServers: ActiveServers{
			HTTP: map[string]LoadBalancerServers{},
			TCP:  map[string]LoadBalancerServers{},
		},
	}

	// preload config
	c.ReadConfig()

	t := time.NewTicker(20 * time.Second)

	go func() {
		for {
			select {
			case <-t.C:
				c.ReadConfig()
			}
		}
	}()

	return c, nil
}

func (c *InstanceManager) ReadConfig() *InstanceManager {
	cfg, err := os.ReadFile("config/services.yaml")
	if err != nil {
		log.Println("failed to read config file, err: ", err)
		return c
	}

	buf := bytes.NewBuffer(cfg)

	if err := yaml.NewDecoder(buf).Decode(&c.InstanceGroupRegistrar); err != nil {
		log.Println("failed to decode config file, err: ", err)
		return c
	}

	for _, service := range c.InstanceGroupRegistrar.Services {
		if service.ServiceType == ServiceTypeTCP && service.Port == 0 {
			slog.Error("empty port on tcp service", "service", service.GroupName, "service_type", service.ServiceType)
			continue
		}

		lxdClient, err := c.clientPool.Get()
		if err != nil {
			log.Println("error get a client, err: ", err)
			continue
		}

		lxdClient.UseProject(service.ProjectName)

		instances, err := lxdClient.GetInstancesFull(service.InstanceType)
		if err != nil {
			log.Printf("failed to get instances full for [%s]: group [%s], err: %s", service.ProjectName, service.GroupName, err)
			continue
		}

		c.RegisterGroup(service, instances)
	}

	return c
}

func (c *InstanceManager) RegisterGroup(service VirtualInstanceGroup, instances []api.InstanceFull) *InstanceManager {
	switch service.ServiceType {
	case ServiceTypeHTTP:
	case ServiceTypeTCP:
	default:
		service.ServiceType = ServiceTypeHTTP
	}

	addresses := make([]string, 0, len(instances))

	for _, instance := range instances {
		if !strings.HasPrefix(instance.Name, service.InstancePrefix) {
			continue
		}

		if !instance.IsActive() {
			continue
		}

		eth0, ok := instance.State.Network["eth0"]
		if !ok {
			continue
		}

		inet4 := ""

		for _, address := range eth0.Addresses {
			if address.Family != "inet" {
				continue
			}

			inet4 = address.Address
		}

		if inet4 == "" {
			continue
		}

		if service.ServiceType == ServiceTypeHTTP {
			if !(service.Port == 0 || service.Port == defaultHTTPPort) {
				inet4 = fmt.Sprintf("http://%s:%d", inet4, service.Port)
			}
		} else if service.ServiceType == ServiceTypeTCP {
			inet4 = fmt.Sprintf("%s:%d", inet4, service.Port)
		}

		addresses = append(addresses, inet4)
	}

	if service.ServiceType == ServiceTypeHTTP {
		c.ActiveServers.HTTP[service.GroupName] = LoadBalancerServers{
			Servers: addresses,
			Options: service.LBOptions,
		}
	} else {
		c.ActiveServers.TCP[service.GroupName] = LoadBalancerServers{
			Servers: addresses,
			Options: service.LBOptions,
		}
	}

	return c
}
