package client

import (
	"context"
	"errors"
	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"log"
	"math/rand"
	"os"
	"slices"
	"sync"
	"time"
)

var ErrPoolExhausted = errors.New("lxd client: pool exhausted")

type PoolConfig struct {
	MaxPoolSize           int           `yaml:"max_pool_size"`
	MaxIdleConnections    int           `yaml:"max_idle_connections"`
	IdleConnectionTimeout time.Duration `yaml:"idle_connection_timeout"`
}
type ConnectionPool struct {
	idleConnections []*LXDClient
	usedConnections []*LXDClient
	mutex           sync.Mutex
	config          PoolConfig
}

// LXDInstanceServer type alias

//go:generate mockgen -source=pool.go -destination=../gen/mock/client/pool.go
type LXDServer interface {
	GetInstance(name string) (instance *api.Instance, ETag string, err error)
	GetInstanceFull(name string) (instance *api.InstanceFull, ETag string, err error)
	UseProject(name string) (client lxd.InstanceServer)
	GetInstancesFullWithFilter(instanceType api.InstanceType, filters []string) (instances []api.InstanceFull, err error)

	GetInstancesFull(instanceType api.InstanceType) (instances []api.InstanceFull, err error)
}

// LXDClient wrapper, to extend the instance server capability.
type LXDClient struct {
	LXDServer
	id       int
	lastUsed time.Time
}

func (c *LXDClient) updateLastUsed() bool {
	c.lastUsed = time.Now()
	return true
}

func create() (lxd.InstanceServer, error) {
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

	return lxd.ConnectLXDWithContext(context.Background(), serverURL, &lxd.ConnectionArgs{
		InsecureSkipVerify: true,
		TLSClientCert:      string(certFile),
		TLSClientKey:       string(keyFile),
	})
}

func (c *ConnectionPool) Get() (*LXDClient, error) {
	if len(c.idleConnections) == 0 {
		if err := c.open(rand.Int()); err != nil {
			return nil, ErrPoolExhausted
		}
	}

	c.mutex.Lock()
	var conn *LXDClient
	conn, c.idleConnections = c.idleConnections[0], c.idleConnections[1:]

	c.usedConnections = append(c.usedConnections, conn)

	c.mutex.Unlock()

	return conn, nil
}

func (c *ConnectionPool) Release(conn *LXDClient) error {
	c.mutex.Lock()
	c.usedConnections = nil
	slices.DeleteFunc(c.usedConnections, func(client *LXDClient) bool {
		return client.id == conn.id
	})
	c.idleConnections = append(c.idleConnections, conn)
	c.mutex.Unlock()
	return nil
}

func NewClientConnectionPool(config PoolConfig) *ConnectionPool {
	if config.MaxIdleConnections == 0 {
		config.MaxIdleConnections = 3
	}

	if config.MaxPoolSize == 0 {
		config.MaxPoolSize = 5
	}

	if config.IdleConnectionTimeout == 0 {
		config.IdleConnectionTimeout = 1 * time.Minute
	}

	idleConnections := make([]*LXDClient, 0, config.MaxIdleConnections)

	client := &ConnectionPool{
		idleConnections: idleConnections,
		config:          config,
	}

	for i := 0; i < config.MaxPoolSize; i++ {
		if err := client.open(i); err != nil {
			continue
		}
	}

	// unused idle connections collector
	// to prevent memory leak.
	go func() {
		for {
			client.clearUnusedConnections()
			time.Sleep(10 * time.Second)
		}
	}()

	return client
}

func (c *ConnectionPool) open(id int) error {
	conn, err := create()
	if err != nil {
		return err
	}

	c.idleConnections = append(c.idleConnections, &LXDClient{
		id:        id,
		LXDServer: conn,
	})

	return nil
}

func (c *ConnectionPool) clearUnusedConnections() {
	if len(c.idleConnections) > c.config.MaxIdleConnections {
		c.idleConnections = c.idleConnections[:c.config.MaxIdleConnections]
	}
}
