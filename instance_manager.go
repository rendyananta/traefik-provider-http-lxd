package main

import (
	"bytes"
	"encoding/json"
	"github.com/canonical/lxd/shared/api"
	"log"
	"os"
	"time"
	"traefik-http-lxd-provider/client"
)

type VirtualInstanceGroup struct {
	GroupName      GroupName
	ProjectName    ProjectName
	InstancePrefix string
	InstanceType   api.InstanceType
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
		worker:     worker,
		clientPool: pool,
	}

	go func() {
		for {
			c.ReadConfig()
			time.Sleep(10 * time.Second)
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

	var registrar VirtualInstanceGroupRegistrar

	if err := json.NewDecoder(buf).Decode(&registrar); err != nil {
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

		instances, err := lxdClient.GetInstancesFullWithFilter(service.InstanceType, []string{""})
		if err != nil {
			log.Printf("failed to get instances full for [%s]: group [%s], err: %s", service.ProjectName, service.GroupName, err)
			continue
		}

		//TODO: register to map that save the instance states.
		log.Println(instances)
	}

	return c
}

func (c *InstanceManager) RegisterGroup(group VirtualInstanceGroup, instances []api.InstanceFull) *InstanceManager {
	c.ActiveServices[group.GroupName] = make([]string, 0, len(instances))

	for _, instance := range instances {
		if !instance.IsActive() {
			continue
		}

		ip, ok := instance.Devices["eth0"]
		if !ok {
			continue
		}

		c.ActiveServices[group.GroupName] = append(c.ActiveServices[group.GroupName], "")
	}

	return c
}
