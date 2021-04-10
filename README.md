# Bgp-exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/MikeGoltsov/bgp-exporter)](https://goreportcard.com/report/github.com/MikeGoltsov/bgp-exporter)

This is a simple, lightweight exporter of BGP routes via HTTP for Prometheus.

## Getting Started

Run from command-line:

```bash
./bgp-exporter [flags]
```

The exporter supports two configuration ways: environment variables that take precedence over configuration file.

The configuration file is specified with the flag -c filename. Yaml and ini formats accepted.

As for available config options and equivalent environment variables, here is a list:

|     environment variable      |      config file    | cli flag |                       description                  |        default        |
| ----------------------------- | ------------------- | -------- | -------------------------------------------------- | --------------------- |
| BGPEXP_ASN                    | asn                 | -a       | AS number of exporter                              | 64512                 |
| BGPEXP_ROUTER_ID              | router-id           |          | RID of exporter                                    | 1.1.1.1               |
| BGPEXP_LISTEN_ADDRES          | listen_address      | -l       | IPv4 address of interface where bgp proccess listen| 0.0.0.0               |
| BGPEXP_METRICS_PORT           | metrics-port        |          | Port to listen on for HTTP requests                | 9179                  |
| BGPEXP_DELETE_ON_DISCONNECT   | delete_on_disconnect|          | Remove metrics of disconnected BGP peer            | False                 |
| BGPEXP_LOG_LEVEL              | log-level           |          | Log level                                          | Info                  |


## Metrics

|                name                |                     description                     |
| ---------------------------------- | --------------------------------------------------- |
| bgp_route                          | Route announced by neighbour                        |
| bgp_route_changes                  | Total number of changes (updates,withdraws) of route|
| bgp_neighbour_announced_total      | Total number of of routes announced by neighbour    |
| bgp_local_asn                      | ASN of exporter                                     |

### Systemd

The best way to use the provided service is to run the exporter with systemd   
