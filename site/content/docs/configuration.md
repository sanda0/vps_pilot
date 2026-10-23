---
title: Configuration
weight: 50
description: The hub's .env settings, its command-line flags, what lives in the data directory, and how to run it behind HTTPS.
---

The hub is configured with a `.env` file and a few command-line flags. The
agent has its own options, listed on the [Agent](/docs/agent/#options) page.

## The .env file

Start from the example:

```sh
cd server
cp .env.example .env
```

The hub reads `.env` from the directory it's started in. If the file is
missing it logs a warning and uses the environment only. Real environment
variables always win over values in the file.

| Variable | Default | Description |
| --- | --- | --- |
| `TOKEN_SECRET` | built-in, insecure | Signs login sessions (JWT). Set a long random value: `openssl rand -hex 32`. |
| `TOKEN_LIFESPAN` | `60` | Session lifetime in minutes. The example file sets `10080` (7 days). |
| `COOKIE_SECURE` | automatic | The session cookie's `Secure` flag. Unset: on when the request came over HTTPS, directly or through a proxy that sets `X-Forwarded-Proto`. Set `true` or `false` to force it. |
| `DB_PATH` | `./data` | Directory for the databases and the agent TLS certificate. A leading `~` means your home directory. |
| `TCP_SERVER_PORT` | `55001` | Agent channel port. The `-agent-port` flag overrides it. |
| `MAIL_HOST` | none | SMTP server for email alerts. |
| `MAIL_PORT` | none | SMTP port. The example file sets `587`. |
| `MAIL_USERNAME` | none | SMTP user. |
| `MAIL_PASSWORD` | none | SMTP password. |
| `MAIL_FROM_ADDRESS` | none | Sender address, for example `noreply@example.com`. |

{{< callout warning >}}
If `TOKEN_SECRET` is empty, anyone who reads the source code can forge a
login. The hub logs a warning when that happens. Set it before you expose the
dashboard.
{{< /callout >}}

Email alerts need all five `MAIL_` settings. If one is missing, the hub skips
email and logs why.

## Command-line flags

| Flag | Default | Description |
| --- | --- | --- |
| `-port` | `8080` | Dashboard and API port. |
| `-agent-port` | `55001` | Agent channel (TLS) port. Falls back to `TCP_SERVER_PORT`. |
| `-agent-tls-cert`, `-agent-tls-key` | self-signed in `$DB_PATH/tls` | Your own certificate and key for the agent channel. Set both or neither. |
| `-migrate` | | Apply database migrations and exit. |
| `-create-superuser` | | Create a login (email and password) and exit. |
| `-create-makefile` | | Write a `makefile` with hub tasks to the current directory and exit. |

```sh
./vps_pilot -port 3000            # dashboard on :3000
./vps_pilot -agent-port 6000      # agent channel on :6000
./vps_pilot -migrate              # migrate and exit
./vps_pilot -create-superuser     # add a login
```

Migrations also run automatically every time the hub starts.

## The data directory

Everything the hub keeps lives in `DB_PATH`:

| Path | Contents |
| --- | --- |
| `operational.db` | Logins, nodes, alerts, projects, join tokens, scan settings. |
| `timeseries.db` | CPU, memory and network history. |
| `tls/agent-hub.crt`, `tls/agent-hub.key` | The agent channel certificate, created on first start. |

Back up the whole directory. Losing `tls/` changes the certificate
fingerprint, and then every agent refuses to connect until you re-enroll it.

Metrics are kept for **7 days**. Once an hour the hub deletes anything older.

## Run the hub in production

Serve the dashboard over HTTPS with a reverse proxy in front of port `8080`.
The metrics charts use a WebSocket (`/api/v1/nodes/ws/system-stat`), so the
proxy must pass WebSocket upgrades. Caddy does that out of the box:

```text
vps.example.com {
	reverse_proxy 127.0.0.1:8080
}
```

The agent channel on `55001` is TLS already and doesn't go through the proxy.
Open it to your servers only, if your firewall allows that.

A minimal systemd unit for the hub. `WorkingDirectory` matters, because that's
where the hub looks for `.env` and, by default, `./data`:

```ini
# /etc/systemd/system/vps-pilot.service
[Unit]
Description=VPS Pilot hub
After=network-online.target
Wants=network-online.target

[Service]
User=vpspilot
WorkingDirectory=/opt/vps-pilot
ExecStart=/opt/vps-pilot/vps_pilot
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

{{< callout soon >}}
A Docker Compose setup is planned.
{{< /callout >}}
