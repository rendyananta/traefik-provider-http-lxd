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

type VirtualInstanceGroupRegistrar = map[ProjectName]map[GroupName]VirtualInstanceGroup

type InstanceManager struct {
	clientPool LXDClientPool
	worker     GoroutineWorker
	GroupMap   VirtualInstanceGroupRegistrar
}

func NewInstanceManager(worker GoroutineWorker, pool LXDClientPool) (*InstanceManager, error) {
	c := &InstanceManager{
		worker:     worker,
		clientPool: pool,
		GroupMap:   make(VirtualInstanceGroupRegistrar),
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
	cfg, err := os.ReadFile("config/instance-groups.yaml")
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

	for projectName, virtualInstanceGroups := range registrar {
		lxdClient, err := c.clientPool.Get()
		if err != nil {
			log.Println("error get a client, err: ", err)
			continue
		}

		lxdClient.UseProject(projectName)

		for _, group := range virtualInstanceGroups {
			instances, err := lxdClient.GetInstancesFullWithFilter(group.InstanceType, []string{""})
			if err != nil {
				log.Printf("failed to get instances full for [%s]: group [%s], err: %s", projectName, group.GroupName, err)
				continue
			}

			//TODO: register to map that save the instance states.
			log.Println(instances)
		}
	}

	return c
}

func (c *InstanceManager) RegisterGroup(projectName ProjectName, group VirtualInstanceGroup, instances []api.InstanceFull) *InstanceManager {
	_, ok := c.GroupMap[projectName]
	if !ok {
		c.GroupMap[projectName] = map[GroupName]VirtualInstanceGroup{}
	}

	c.GroupMap[projectName][group.GroupName] = VirtualInstanceGroup{
		GroupName:      group.GroupName,
		ProjectName:    projectName,
		InstancePrefix: group.InstancePrefix,
		Members:        make([]InstanceInfo, 0),
	}

	return c
}
