---
title: Security
weight: 70
description: How agents authenticate, what the hub stores, what a compromised hub can and can't do, and how to run VPS Pilot safely.
---

## The agent channel

- **TLS 1.3 only**, on both the hub and the agent.
- The hub uses a self-signed certificate (ECDSA P-256), created on first
  start in `$DB_PATH/tls/`, or your own via `-agent-tls-cert` and
  `-agent-tls-key`.
- The agent doesn't trust certificate authorities or check host names.
  It **pins the SHA-256 fingerprint** of the hub's certificate, which it
  learns from the join token. The TLS handshake signature is still verified,
  so only the holder of that certificate's private key can talk to the agent.
- A connection has 10 seconds to authenticate. Until it does, the hub ignores
  every message except `enroll` and `connected`.
- Frames larger than 1 MiB are rejected and the connection is dropped.
- Rejections are deliberately vague (`invalid credentials`), so they don't
  reveal which part was wrong.

## Credentials

| Credential | What it is | Lifetime | Stored on the hub as |
| --- | --- | --- | --- |
| Join token | `vpsp1.<secret>.<fingerprint>` | One use, 1 hour | SHA-256 of the secret |
| Node key | 32 random bytes | Until revoked or re-enrolled | SHA-256, compared in constant time |
| Agent ID | Random UUID | Per enrollment | Plain |

The agent sends only the secret part of the join token. It keeps its agent
ID, node key and the pinned fingerprint in its state file, readable by its
owner only (mode `0600`).

Nodes are identified by agent ID, never by IP address. Revoking a node's key
in the dashboard disconnects it at once, and a node can only have one live
connection.

## What the dashboard can do on a server

The hub can send an agent the project scan settings, "rescan now", a request
to run or cancel a project command, a request to list or read the files in
a project's declared log folders, and a request to back up a project now. It
can't read any other file or open a shell.

Running commands follows one rule: **the hub sends command names, never
command strings.** The agent reads the project's `config.vpspilot.json` again
and runs only what's written there. Even a compromised hub can only trigger
commands the server owner already wrote into a file on that server.

The agent also decides who a command runs as: the owner of
`config.vpspilot.json`, never automatically the agent's own user (often root).
It refuses a config that is a symlink or writable by its group or by others,
so nobody else can put a command in someone's file. Every run is recorded with
the dashboard user who started it. See [Run a command](/docs/projects/#run-a-command).

Reading logs follows the same rules. The agent re-reads the config, reads as
its owner, and only opens files that resolve, after symlinks, inside a folder
listed in `logs`, which must itself resolve inside the project directory. The
hub sends the file's path as the agent listed it and stores none of the
contents. See [Read logs](/docs/projects/#read-logs).

Backups follow them too. The agent backs up only what `backups` declares, as
the config file's owner, and only paths that resolve inside the project
directory. The zip stays on the server and in the project's bucket; the hub
only records the outcome. The S3 keys and database password are read from the
env file on the server: the config holds only their variable names, so the
hub never sees them. See [Back up](/docs/projects/#back-up).

Manual project paths follow the same idea. The agent only accepts them inside
its own project directories (after resolving `..` and symlinks), unless the
server owner starts it with `--allow-any-project-path`.

## The dashboard

- Passwords are hashed with bcrypt.
- A login is a JWT (HS256, signed with `TOKEN_SECRET`) in an `HttpOnly`,
  `SameSite=Lax` cookie, marked `Secure` when the request came over HTTPS.
- Every login has full access. There are no roles yet.

{{< callout soon >}}
Multiple users with roles are planned.
{{< /callout >}}

## Checklist

- Set `TOKEN_SECRET` to a long random value (`openssl rand -hex 32`). If it's
  empty, the hub signs logins with a built-in default that anyone can find.
- Serve the dashboard over HTTPS through a reverse proxy, and don't expose
  port `8080` directly.
- Restrict port `55001` to your servers' IP addresses if your firewall allows
  it.
- Back up the data directory, including `tls/`. If the certificate is lost,
  every agent has to be re-enrolled.
- Keep `.env` private: it holds `TOKEN_SECRET` and the SMTP password.

## Known limitations

- `-create-superuser` prints the password back to the terminal once the user
  is created.
- The hub doesn't verify the SMTP server's TLS certificate when it sends
  email alerts.
- Alert webhook URLs are stored in the hub's database in plain text, like the
  rest of the alert settings.
