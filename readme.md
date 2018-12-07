# Balance exporter for Selectel service

The selectel balance exporter allows exporting balance for each service in [selectel](https://selectel.ru) to [prometheus](https://prometheus.io)

## How it works
Exporter request balance every hour (by default) and store it value in memory.
When prometheus request metrics, exporter send balance value from memory.

## Configuration
You must set environment variable:

* `SELECTEL_API_KEY` - your api token. Api key can be generated at page https://my.selectel.ru/profile/apikeys

## Command-line flags

* `listen-address` - The address to listen on for HTTP requests. (Default: `0.0.0.0:9600`)
* `interval` - Interval (in seconds) for balance requests. (Default: `3600`)

## Running with docker

```sh
docker run \
    -e SELECTEL_API_KEY=<your-token> \
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
      SELECTEL_API_KEY: <your-token>
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
SELECTEL_EXPORTER_VERSION=v0.1.7-beta.1
SELECTEL_EXPORTER_PLATFORM=linux
SELECTEL_EXPORTER_ARCH=amd64
SELECTEL_API_KEY=<your-token>
```

Download release:
```sh
wget https://github.com/xxxcoltxxx/selectel-balance-exporter/releases/download/${SELECTEL_EXPORTER_VERSION}/selectel_balance_exporter_${SELECTEL_EXPORTER_VERSION}_${SELECTEL_EXPORTER_PLATFORM}_${SELECTEL_EXPORTER_ARCH}.tar.gz
tar xvzf selectel_balance_exporter_${SELECTEL_EXPORTER_VERSION}_${SELECTEL_EXPORTER_PLATFORM}_${SELECTEL_EXPORTER_ARCH}.tar.gz
mv ./selectel_balance_exporter_${SELECTEL_EXPORTER_VERSION}_${SELECTEL_EXPORTER_PLATFORM}_${SELECTEL_EXPORTER_ARCH} /usr/local/bin/selectel_balance_exporter
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
ExecStart=/usr/local/bin/selectel_balance_exporter

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
