---
title: Alerts
weight: 40
description: Get a Discord, Slack or email message when a server's CPU or memory crosses a threshold.
---

An alert watches one metric on one node. The hub checks it against every
stat report from the agent (every five seconds by default) and notifies
every channel you filled in.

## Create an alert

1. Open the node and switch to the **Alerts** tab.
2. Choose **Create Alert**.
3. Fill in the form:
   - **Metric**: CPU or Memory.
   - **Threshold (%)**: the alert fires when the value goes *above* it. For
     CPU, the value is the average across all cores.
   - **Duration (in minutes)**: how long to wait before notifying again
     while the value stays above the threshold. With `10`, you get at most
     one message every ten minutes.
   - **Email**, **Discord Webhook**, **Slack Webhook**: fill in any
     combination. Each one that's set gets the message.
   - **Enabled**: alerts can be switched off without deleting them.

Each message carries the node's name and IP, the metric, the current value,
the threshold and the time.

{{< callout warning >}}
Set a duration of at least 1. With `0`, a node that stays over the threshold
sends a message on every stat report, every five seconds. The wait between
messages is kept in memory, so it also starts over when the hub restarts.
{{< /callout >}}

{{< callout soon >}}
Network alerts are in the form, but don't work as intended yet: the hub
compares traffic against a field the form doesn't set, instead of the send
and receive thresholds. Disk alerts, and a history of past alerts, are
planned too.
{{< /callout >}}

## Discord

1. In Discord, open **Server Settings → Integrations → Webhooks**.
2. Choose **New Webhook**, pick the channel, and copy the webhook URL.
3. Paste it into the alert's **Discord Webhook** field.

## Slack

1. Create a Slack app (or use an existing one) and turn on **Incoming
   Webhooks**.
2. Choose **Add New Webhook to Workspace** and pick the channel.
3. Copy the webhook URL into the alert's **Slack Webhook** field.

## Email

Email goes through your own SMTP server, set in the hub's `.env`:

```sh
MAIL_HOST=smtp.example.com
MAIL_PORT=587
MAIL_USERNAME=alerts@example.com
MAIL_PASSWORD=your-smtp-password
MAIL_FROM_ADDRESS=noreply@example.com
```

All five are required; if one is missing the hub skips email and logs
`email configuration is incomplete`. Restart the hub after editing `.env`,
then put the recipient's address in the alert's **Email** field. Gmail and
most other providers need an app password rather than your login password.

See [Configuration](/docs/configuration/#the-env-file) for the other hub
settings.
