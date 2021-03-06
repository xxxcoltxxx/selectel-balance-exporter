# Balance exporter for Selectel service

The selectel balance exporter allows exporting balance for each service in [selectel](https://selectel.ru) to [prometheus](https://prometheus.io)

## How it works
Exporter request balance every hour (by default) and store it value in memory.
When prometheus request metrics, exporter send balance value from memory.

## Configuration
You must set environment variable:

* `SELECTEL_API_KEY` - your api key. Api key can be generated at page https://my.selectel.ru/profile/apikeys

## Command-line flags

* `listen-address` - The address to listen on for HTTP requests. (Default: `0.0.0.0:9600`)
* `interval` - Interval (in seconds) for balance requests. (Default: `3600`)
* `retry-interval` - Interval (in seconds) for load balance when errors. (Default: `10`)
* `retry-limit` - Count of tries when error. (Default: `10`)

## Metrics example
```
# HELP balance_selectel Balance for service in selectel account
# TYPE balance_selectel gauge
balance_selectel{service="primary"} 10000.54
balance_selectel{service="storage"} 20000.52
balance_selectel{service="vmware"} 0
balance_selectel{service="vpc"} 300000.47
```


## Running with docker

```sh
docker run \
    -e SELECTEL_API_KEY=<your-key> \
    -p 9600:9600 \
    --restart=unless-stopped \
    --name selectel-balance-exporter \
    -d \
    xxxcoltxxx/selectel-balance-exporter
```

## Running with docker-compose

Create configuration file. For example, file named `docker-compose.yaml`:

```yaml
version: "3"

services:
  selectel-balance-exporter:
    image: xxxcoltxxx/selectel-balance-exporter
    restart: unless-stopped
    environment:
      SELECTEL_API_KEY: <your-key>
    ports:
      - 9600:9600
```

Run exporter:
```sh
docker-compose up -d
```

Show service logs:
```sh
docker-compose logs -f selectel-balance-exporter
```

## Running with systemctl

Set variables you need:
```sh
SELECTEL_EXPORTER_VERSION=0.2.0
SELECTEL_EXPORTER_PLATFORM=linux
SELECTEL_EXPORTER_ARCH=amd64
SELECTEL_API_KEY=<your-key>
```

Download release:
```sh
wget https://github.com/xxxcoltxxx/selectel-balance-exporter/releases/download/v${SELECTEL_EXPORTER_VERSION}/selectel-balance-exporter_${SELECTEL_EXPORTER_VERSION}_${SELECTEL_EXPORTER_PLATFORM}_${SELECTEL_EXPORTER_ARCH}.tar.gz
tar xvzf selectel-balance-exporter_${SELECTEL_EXPORTER_VERSION}_${SELECTEL_EXPORTER_PLATFORM}_${SELECTEL_EXPORTER_ARCH}.tar.gz
mv ./selectel-balance-exporter /usr/local/bin/selectel-balance-exporter
```

Add service to systemctl. For example, file named `/etc/systemd/system/selectel_balance_exporter.service`:
```sh
[Unit]
Description=Selectel Balance Exporter
Wants=network-online.target
After=network-online.target

[Service]
Environment="SELECTEL_API_KEY=${SELECTEL_API_KEY}"
Type=simple
ExecStart=/usr/local/bin/selectel-balance-exporter

[Install]
WantedBy=multi-user.target
```

Reload systemctl configuration and restart service
```sh
systemctl daemon-reload
systemctl restart selectel_balance_exporter
```

Show service status:
```sh
systemctl status selectel_balance_exporter
```

Show service logs:
```sh
journalctl -fu selectel_balance_exporter
```
