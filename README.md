# Bgp-exporter

This is a simple, lightweight exporter of BGP routes via HTTP for Prometheus.

## Getting Started

Run from command-line:

```bash
./bgp-exporter [flags]
```

The exporter supports two configuration ways: environment variables that take precedence over configuration file.

As for available config options and equivalent environment variables, here is a list:

|     environment variable      |      config file    |                       description                  |        default        |
| ----------------------------- | ------------------- | -------------------------------------------------- | --------------------- |
| BGPEXP_ASN                    | asn                 | AS number of exporter                              | 64512                 |
| BGPEXP_ROUTER_ID              | router_id           | RID of exporter                                    | 1.1.1.1               |
| BGPEXP_LISTEN_ADDR            | listen_address      | IPv4 address of interface where bgp proccess listen| 0.0.0.0               |
| BGPEXP_METRICS_PORT           | metrics_port        | Port to listen on for HTTP requests                | 9179                  |
| BGPEXP_DELETE_ON_DISCONNECT   | delete_on_disconnect| Remove metrics of disconnected BGP peer            | False                 |
| BGPEXP_LOG_LEVEL              | log_level           | Log level                                          | Info                  |


## Metrics

|                name                |                     description                     |
| ---------------------------------- | --------------------------------------------------- |
| bgp_route                          | Route announced by neighbour                        |
| bgp_route_changes                  | Total number of changes (updates,withdraws) of route|
| bgp_local_asn                      | ASN of exporter                                     |


### Systemd

The best way to use the provided service is to run the exporter with systemd   
