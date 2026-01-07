# socks5proxy

A basic SOCKS5 proxy server written in Go.

## Build

```
go build -o socks5proxy .
```

## Usage

```
./socks5proxy [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:1080` | Address and port to listen on |
| `-allowedIPs` | (empty) | Comma-separated list of allowed client IPs. All IPs allowed if empty |

## Examples

```
# Listen on default port 1080
./socks5proxy

# Listen on custom port
./socks5proxy -addr :9999

# Restrict to specific IPs
./socks5proxy -allowedIPs 127.0.0.1,10.0.0.5

# Combined
./socks5proxy -addr :8080 -allowedIPs 192.168.1.100
```

## Test

```
curl --socks5 127.0.0.1:1080 http://example.com
```
