[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/dombyte/ipgate)](https://goreportcard.com/report/github.com/dombyte/ipgate)
[![GitHub go.mod Go version of a Go module](https://img.shields.io/github/go-mod/go-version/dombyte/ipgate.svg)](https://github.com/dombyte/ipgate)
# IPBlocker

A high-performance IP blocking system in Go with caching and flexible configuration.

## Features

- **Blocklist Management**: Support for local and remote blocklists
- **Whitelist Management**: Support for local and remote whitelists
- **Cache**: Temporary caching of blocked and allowed IPs with configurable TTL
- **API Endpoints**: HTTP endpoints for IP blocking and health checks
- **File Watching**: Automatic reloading on changes to local files
- **Remote Blocklists**: Regular updating of remote blocklists via cron
- **IPv4/IPv6 Support**: Full support for both IPv4 and IPv6 addresses
- **CIDR Notation**: Support for CIDR blocks in blocklists and whitelists

> [!IMPORTANT]  
> The current Cache implemenation is broken. When enabled a high number of concurrent requests will increase response time on first request (Cache Miss).\




## Build

### Prerequisites

- Go 1.25+
- Docker (optional, for container operation)

### From Source

```bash
git clone https://github.com/dombyte/ipgate.git
cd ipgate
go build -o ipgate ./cmd/ipgate
```

### With Docker

```bash
docker-compose up --build
```

## Usage
### Example Configuration
There is an example docker-compose configuration file in the `example` directory that you can use as a starting point. It includes a sample configuration file and demonstrates how to run the application with Docker Compose. You can modify the configuration file as needed and then use the provided docker-compose file to run the application.


### Binary 
```bash
./ipgate --config ./config/config.yaml
```
### With Docker

```bash
docker run -d --name ipgate -p 8080:8080 \
  -v ./config:/app/config \
  ghcr.io/dombyte/ipgate:latest
```
### With Docker Compose

```yaml
---
services:
  ipgate:
    image: ghcr.io/dombyte/ipgate:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config 
    command:
      - "--log.level=DEBUG"
      - "--config=/app/config/config.yaml"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "httpcheck", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```



## Configuration

Configuration is done via `config/config.yaml`:

```yaml
port: "8080"
error_page: "/app/templates/error.html"
error_format: "html" # or "json"
status_allowed: 200
status_denied: 403
debug_endpoint: false # Set to true to enable debug endpoints
watch_files_enabled: true # Set to false to disable file watching

headers:
  client_ip_header: "X-Forwarded-For"
  host_header: "X-Forwarded-Host"
  uri_header: "X-Forwarded-Uri"
  method_header: "X-Forwarded-Method"
  proto_header: "X-Forwarded-Proto"

cache:
  enabled: true
  ttl: 300 # seconds
  max_entries: 100000
  prune_interval: 60 # seconds
  shard_count: 64
  prune_on_get: false
  write_buffer_size: 0
  auto_clear_on_change: true

blocklist_max_size: 10485760 # bytes

whitelist_files:
  - path: "/app/config/whitelists/whitelist.txt"
  - path: "/app/config/whitelists/whitelist2.txt"

whitelist_remotes:
  - url: "https://ipv64.net/whitelists/whitelist.txt"
  - url: "https://ipv64.net/whitelists/whitelist2.txt"
    cron: "0 * * * *"

blacklist_files:
  - path: "/app/config/blocklists/blocklist.txt"
  - path: "/app/config/blocklists/blocklist2.txt"

blacklist_remotes:
  - url: "https://ipv64.net/blocklists/ipv64_blocklist_all.txt"
  - url: "https://ipv64.net/blocklists/ipv64_blocklist_all.txt"
    cron: "0 * * * *"
```

> [!TIP]
> You can specify multiple local blocklist and whitelist files as well as multiple remote blocklists and whitelists. The application will aggregate all entries from the specified sources to create the final blocklist and whitelist used for IP blocking decisions.


### CLI Arguments

The application supports the following CLI arguments:

```bash
./ipgate --config ./config/config.yaml --log.level INFO
```

- `--config`: Path to configuration file (if not provided, uses **default lookup order**)
- `--log.level`: Log level (DEBUG, INFO, WARN, ERROR) - default: INFO

**Default Configuration Lookup Order**

1. [WORK_DIR]/ipgate.yml or .yaml
2. [WORK_DIR]/config.yml or .yaml
3. [WORK_DIR]/config/ipgate.yml or .yaml
4. [WORK_DIR]/config/config.yml or .yaml

## API Endpoints

### Main Functionality

- `GET /bouncer` - Main blocker endpoint (with cache support)
- `GET /health` - Health check

### Debug Endpoints (Optional)

The debug endpoints are **disabled by default** for security. To enable them, set `debug_endpoint: true` in your configuration file.

Available debug endpoints when enabled:
- `GET /debug/clear-cache` - Clear cache
- `GET /debug/cache-dump` - Show cache contents
- `GET /debug/config` - Show current configuration

> [!IMPORTANT]  
> Enabling debug endpoints can increase memory usage due to maintaining a second copy of the blocklist/whitelist data for debugging purposes. Use with caution in production environments.


## Usage Examples

### Test IP

```bash
curl -X GET "http://localhost:8080/bouncer" \
  -H "X-Forwarded-For: 198.51.100.1"
```

### Get Configuration

```bash
curl http://localhost:8080/debug/config
```

### Clear Cache

```bash
curl http://localhost:8080/debug/clear-cache
```

## Blocklist and Whitelist Files

### Blocklist File Format

```txt
# IPBlocker Blocklist
# Format: Individual IPs or CIDR notation
# Empty lines and comments are ignored

# Example individual IPs
198.51.100.1
203.0.113.1

# Example CIDR blocks
198.51.100.0/24
203.0.113.0/24

# IPv6 examples
2001:db8::1
2001:db8::/32
```

### Whitelist File Format

```txt
# IPBlocker Whitelist
# Format: Individual IPs or CIDR notation
# Empty lines and comments are ignored

# Example individual IPs
198.51.100.1
203.0.113.1

# Example CIDR blocks
198.51.100.0/24
203.0.113.0/24

# IPv6 examples
2001:db8::1
2001:db8::/32
```

## Caching

The IPBlocker implements an intelligent caching system for both blocked and allowed IPs.

### Cache Configuration

```yaml
cache:
  enabled: true                    # Enable/disable cache (default: true)
  ttl: 300                         # Cache entry lifetime in seconds (default: 300)
  max_entries: 100000              # Maximum number of entries in cache (default: 100000)
  prune_interval: 60               # How often to prune expired entries (seconds, default: 60)
  shard_count: 64                  # Number of shards for concurrent access (default: 64)
  prune_on_get: false              # Prune expired entries on cache get (higher CPU but lower memory, default: false)
  write_buffer_size: 0             # Batch write buffer size (0 for disabled, default: 0)
  auto_clear_on_change: true      # Clear cache when blocklists change (default: true)
```

### Cache Behavior

1. **First Request**: IP is checked against blocklists/whitelist
   - If blocked → Added to cache with status DENY
   - If allowed → Added to cache with status ALLOW
   - Timestamp is set to current time

2. **Subsequent Requests**: IP is checked in cache first
   - If in cache and not expired → Served from cache (fast)
   - If expired or not in cache → Check blocklists/whitelist again

3. **TTL Expiration**: After TTL seconds, cache entry expires
   - Next request triggers fresh blocklist check
   - New cache entry created with updated status

> [!IMPORTANT]  
> TTL is NOT reset on cache hits. This prevents stale cache entries from staying in cache forever.


## File Watching

Local blocklist and whitelist files are automatically watched for changes (if `watch_files_enabled: true`). When a file changes, the application will:

1. Reload the file contents
2. Reload the remote
3. Update the internal blocklist/whitelist data
4. Clear the cache (if `auto_clear_on_change: true`)

This allows for dynamic updates without requiring a restart.

## Remote Blocklist Updates
Remote blocklists and whitelists are updated on a regular schedule defined by cron expressions in the configuration. The application will fetch the remote lists at the specified intervals and update the internal blocklist/whitelist data accordingly. It also will reload the local files to ensure that any changes are reflected in the blocklist/whitelist data.

### Cron Schedule

The cron schedule is defined using standard cron syntax. For example, `0 * * * *` means the remote list will be updated every hour at the top of the hour.
It uses the `github.com/robfig/cron/v3` package for parsing and scheduling cron jobs, which supports a wide range of cron expressions, like `@hourly`, `@daily`, etc., in addition to the standard cron syntax.

### Multiple Remote Lists: 
You can configure multiple remote blocklists and whitelists, each with its own cron schedule. This allows for flexible updating based on the needs of your application.
```yaml

whitelist_remotes:
  - url: "https://example.com/whitelist.txt"
    cron: "0 * * * *" # Update every hour
  - url: "https://example.com/another_whitelist.txt"
  
blacklist_remotes:
  - url: "https://example.com/blocklist.txt"
    cron: "0 * * * *" # Update every hour
  - url: "https://example.com/another_blocklist.txt"
  
```

## Error Templates
The application serves a customizable error page when an IP is blocked. The default template is located at `templates/error.html`. You can modify this file to change the appearance and content of the error page.

**The following template variables are available for use in the error page:**
- `{{.RequestID}}`: A unique identifier for the request 
- `{{.ClientIP}}`: The IP address of the client that was blocked
- `{{.Host}}`: The host header from the request
- `{{.URI}}`: The URI that was requested
- `{{.Method}}`: The HTTP method used in the request
- `{{.Proto}}`: The protocol used in the request (e.g., HTTP/
- `{{.Status}}`: The HTTP status code returned (e.g., 403)
- `{{.BlockReason}}`: The reason for blocking 

> [!TIP]
> You can display any other available header values by using the appropriate header variable (e.g., `{{index .Headers "X-Forwarded-For"}}` for the `X-Forwarded-For` header or `{{index .Headers "User-Agent"}}` for the User-Agent).

## Request Flow
![alt text](./flow.drawio.svg)

## License

IPGate is licensed under the GNU General Public License v3.0. TL;DR — You may copy, distribute and modify the software as long as you track changes/dates in source files. Any modifications to or software including (via compiler) GPL-licensed code must also be made available under the GPL along with build & install instructions. For more information about the license check the [license](./LICENSE) file.

## Support

For questions or issues, please create an issue on GitHub.
