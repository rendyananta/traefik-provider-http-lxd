# Traefik HTTP Provider for LXD and/or Incus.

## Background
This is a simple HTTP app to provide a traefik config that useful for load balancing [LXD](https://documentation.ubuntu.com/lxd/) /  [Incus](https://linuxcontainers.org/incus/docs/main/) instances.
Currently, supports in traefik provider are limited to docker, consul, nomad, etc. (except LXD). 
Initially this repository was an experiments to group LXD instances, which currently not supported by both [LXD](https://documentation.ubuntu.com/lxd/) /  [Incus](https://linuxcontainers.org/incus/docs/main/). [The discourse ](https://discuss.linuxcontainers.org/t/whats-the-progress-of-lxd-instance-group/15679/7) was stopped and the reported issue was invalid since the project moved to the Canonical. This app filling that hole that would be useful for you who need to orchestrate an LXD / Incus instances. As far as we know that the LXD / Incus instances orchestration solution is very limited, so the usage combination of LXD / Incus along with Traefik and Terraform / Tofu should be considerable. 

## Features
1. Gracefully handled.
2. Automatic config discovery. In every configured tick, this app will re-read the config files. This will enable users to leave it after editing config without restarting the HTTP application. This app will update the config in the background.
3. LXD client pool to reduce waiting time on every request to LXD API. This will increase performance.
4. Worker pool to update the provider config for Traefik.
5. Customize the Traefik load balancer options in the HTTP app. 
6. Can be used as an orchestration tool for LXD replica instances.

## How To
To squeeze out all the LXD and Traefik potentials, we need to configure both HTTP Server provider and Traefik to be the orchestration tools. 

### Configure HTTP Server Provider for Traefik
- TBD

### Configure Traefik
- TBD

### Orchestration Example with Terraform / Tofu
- TBD

## What If
There is scenarios what will be happened if something went wrong.
### HTTP Server Starting Up
The HTTP app will always have a delay to provide config data. Trying to fetch the traefik configuration in this state will give an error `423 Status Locked` HTTP status code with locked status. This behaviour was intended to prevent the Traefik store the load balancer configuration with an empty values.

### App Already Started, Fetching Config.
If app already started, but not all the instances discovered, it will wait until all the instances that already defined in config successfully loaded. This will prevent Traefik replacing config changes from the server services that containing an empty load balancer IP address. The HTTP server will keep returning `423 Status Locked`.

### App is down
Traefik will still using the existing configurations on last sync with HTTP Provider. This cache will be used until the HTTP app started again. 

### Traefik is down
As we all know that the Traefik Proxy  also need to be deployed into several instances to prevent a Single Point of Failures. By its [official blog](https://traefik.io/blog/load-balancing-ha-clusters-in-bare-metal-environments/), it is recommended to deploy more than one node as a backup server if the other node fails. 


