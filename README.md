# Lattice

A high-performance API Gateway written in Go that provides authentication, caching, rate limiting, and dynamic configuration management.

## Features

**Core Features**

-   [ ] Dynamic route configuration via Redis
-   [ ] JWT and API key authentication
-   [ ] Response caching with Redis
-   [ ] Distributed rate limiting
-   [x] Reverse proxy to upstream services
-   [x] Automatic retry
-   [ ] Circuit breaking
-   [ ] Load balancing

**Observability**

-   [ ] Prometheus metrics
-   [ ] Grafana dashboards
-   [x] Structured logging (zap)
-   [ ] Distributed tracing
-   [ ] Health checks

**Security**

-   [ ] JWT validation
-   [ ] API key management
-   [ ] Role-based access control
-   [ ] TLS termination
-   [ ] Request validation

## Architecture

```mermaid
graph TD
    Client[Client] --> Lattice[Lattice API Gateway]
    Redis[(Redis)] --> Lattice
    Lattice --> Redis[(Redis)]
    Lattice --> Upstream1[Upstream 1]
    Lattice --> Upstream2[Upstream 2]
    Lattice --> UpstreamN[Upstream N]
    Lattice --> Prometheus[Prometheus]
    Prometheus --> Grafana[Grafana]
```
