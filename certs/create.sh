#!/bin/bash

openssl req -new -nodes -x509 -days 365 -keyout lxd-traefik.key -out lxd-traefik.crt -config target.conf
openssl pkcs12 -keypbe PBE-SHA1-3DES -certpbe PBE-SHA1-3DES -export -in lxd-traefik.crt -inkey lxd-traefik.key -out lxd-traefik.pfx -name "LXD Traefik HTTP Provider"