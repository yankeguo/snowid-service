# winter

A Minimalist Mesh-Native Microservice Framework

## Features

* Include `opentelemetry-go`
    * Support standard `OTEL_` environment variables
    * Using `zipkin` as default exporter
    * Support `TraceContext`, `Baggage` and `B3` propagation
    * Support `otelhttp` instrument
* Include `prometheus/promhttp`
    * Expose at `/debug/metrics`
* Built-in `Readiness Check`
    * Expose at `/debug/ready`
    * Component readiness registration with `App#Check()`
* Built-in `Liveness Check`
    * Expose at `/debug/alive`
    * Cascade `Liveness Check` failure from continuous `Readiness Check` failure
* Built-in `debug/pprof`
    * Expose at `/debug/pprof`
* Easy request binding
    * Unmarshal `Header`, `Query`, `JSON Body` and `Form Body` into any structure with `json` tag

## Donation

See https://guoyk.xyz/donation

## Credits

Guo Y.K., MIT License
