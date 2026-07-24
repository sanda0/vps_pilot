# VPS Pilot C++ Agent

Linux monitoring agent for the VPS Pilot server. It collects per-core CPU,
memory, root-filesystem disk, and non-loopback network usage.

## Requirements

- Linux
- C++20 compiler (GCC 11+ or Clang 14+)
- CMake 3.23+
- Conan 2

## Build

```sh
cd agent
conan profile detect --force
conan install . --output-folder=build --build=missing -s build_type=Release
cmake --preset conan-release
cmake --build --preset conan-release
```

The binary is created at `build/Release/vps-pilot-agent`.

## Run

```sh
cp config.example.json config.json
./build/Release/vps-pilot-agent --config ./config.json
```

Point `server_host` at the VPS Pilot server and ensure its agent TCP port
(default `55001`) is reachable. Set the same long random `agent_token` in the
agent configuration and `AGENT_TOKEN` in the server environment for
authentication.

## Protocol

Messages use a four-byte unsigned big-endian payload length followed by a UTF-8
JSON object. The maximum frame size is 4 MiB.

```json
{
  "type": "sys_stat",
  "node_id": 1,
  "data": {
    "cpu_usage": [8.2, 11.4],
    "mem_usage": 37.5,
    "disk_usage": 52.1,
    "net_sent_ps": 1024,
    "net_recv_ps": 4096
  }
}
```

The first message must be `connected` with system information in `data`. The
server replies with `sys_stat` and the assigned `node_id`.

## systemd

Install the binary and configuration, then copy
`packaging/vps-pilot-agent.service` to `/etc/systemd/system/`. Review its paths
and hardening settings before enabling it.
