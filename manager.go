package main

import (
	"bytes"
	"github.com/canonical/lxd/shared/api"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
	"traefik-http-lxd-provider/client"
)

type VirtualInstanceGroup struct {
	GroupName      GroupName        `yaml:"group_name"`
	ProjectName    ProjectName      `yaml:"project_name"`
	InstancePrefix string           `yaml:"instance_prefix"`
	InstanceType   api.InstanceType `yaml:"instance_type"`
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

type InstanceManager struct {
	clientPool     LXDClientPool
	worker         GoroutineWorker
	GroupMap       VirtualInstanceGroupRegistrar
	ActiveServices map[string][]string
}

func NewInstanceManager(worker GoroutineWorker, pool LXDClientPool) (*InstanceManager, error) {
	c := &InstanceManager{
		worker:         worker,
		clientPool:     pool,
		ActiveServices: map[string][]string{},
	}

	c.ReadConfig()

	return c, nil
}

func (c *InstanceManager) ReadConfig() *InstanceManager {
	cfg, err := os.ReadFile("config/services.yaml")
	if err != nil {
		log.Println("failed to read config file, err: ", err)
		return c
	}

	buf := bytes.NewBuffer(cfg)

	var registrar VirtualInstanceGroupRegistrar

	if err := yaml.NewDecoder(buf).Decode(&registrar); err != nil {
		log.Println("failed to decode config file, err: ", err)
		return c
	}

	for _, service := range registrar.Services {
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

func (c *InstanceManager) RegisterGroup(group VirtualInstanceGroup, instances []api.InstanceFull) *InstanceManager {
	c.ActiveServices[group.GroupName] = make([]string, 0, len(instances))

	for _, instance := range instances {
		if !strings.HasPrefix(instance.Name, group.InstancePrefix) {
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

		c.ActiveServices[group.GroupName] = append(c.ActiveServices[group.GroupName], inet4)
	}

	return c
}
