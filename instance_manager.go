package main

import (
	"bytes"
	"context"
	"encoding/json"
	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"log"
	"os"
	"time"
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

type VirtualInstanceGroupRegistrar = map[ProjectName]map[GroupName]VirtualInstanceGroup

type InstanceManager struct {
	lxd.InstanceServer
	GroupMap      VirtualInstanceGroupRegistrar
	ProjectClient map[ProjectName]lxd.InstanceServer
}

func NewInstanceManager() (*InstanceManager, error) {
	certPath := os.Getenv("CERT_PATH")
	if certPath == "" {
		certPath = "certs/lxd-traefik.crt"
	}

	keyPath := os.Getenv("KEY_PATH")
	if keyPath == "" {
		keyPath = "certs/lxd-traefik.key"
	}

	certFile, err := os.ReadFile(certPath)
	if err != nil {
		log.Fatal(err)
	}

	keyFile, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatal(err)
	}

	serverURL := os.Getenv("LXD_SERVER_URL")
	if serverURL == "" {
		serverURL = "https://localhost:8443"
	}

	serverConn, err := lxd.ConnectLXDWithContext(context.Background(), serverURL, &lxd.ConnectionArgs{
		InsecureSkipVerify: true,
		TLSClientCert:      string(certFile),
		TLSClientKey:       string(keyFile),
	})

	if err != nil {
		return nil, err
	}

	client := &InstanceManager{
		InstanceServer: serverConn,
		GroupMap:       make(VirtualInstanceGroupRegistrar),
	}

	go func() {
		for {
			client.ReadConfig()
			time.Sleep(10 * time.Second)
		}
	}()

	return client, nil
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
		if _, ok := c.ProjectClient[projectName]; !ok {
			newClient := c.InstanceServer.UseProject(projectName)
			if newClient == nil {
				log.Printf("project client for [%s] is nil", projectName)
				continue
			}

			c.ProjectClient[projectName] = newClient
		}

		client := c.ProjectClient[projectName]

		for _, group := range virtualInstanceGroups {
			instances, err := client.GetInstancesFullWithFilter(group.InstanceType, []string{""})
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
