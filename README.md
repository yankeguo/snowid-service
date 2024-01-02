# snowid-service

[![release](https://github.com/yankeguo/snowid-service/actions/workflows/release.yml/badge.svg?branch=main)](https://github.com/yankeguo/snowid-service/actions/workflows/release.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/yankeguo/snowid-service)](https://hub.docker.com/r/yankeguo/snowid-service)

A simple id generation service by snowid

## Usage

### Environment Variable

- `BIND`, address to bind, default to empty
- `PORT`, port to listen, default to `8080`
- `WORKER_ID`, unique worker id, if not set, will guess from hostname (compatible with Kubernetes StatefulSet)
- `EPOCH`, epoch of snow id in UTC, default to `2020-01-01 00:00:00`
- `GRAIN`, snow id grain, default to `1ms`, mimimum to `1ms`
- `LEADING_BIT`, weather to set leading bit, default to `false`

### Invocation

```
GET /healthz

OK
----------
GET /metrics

...Prometheus Metrics...
----------
GET /any/other/path?size=10

[
  "380651065730707456",
  "380651065730707457",
  "380651065730707458",
  "380651065730707459",
  "380651065730707460",
  "380651065730707461",
  "380651065730707462",
  "380651065730707463",
  "380651065730707464",
  "380651065730707465"
]
```

## Credits

GUO YANKE, MIT License
