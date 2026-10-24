package tcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

var lastAlertSentTime map[int32]time.Time

func MontiorAlerts(ctx context.Context, repo *db.Repo, monitorChan chan Msg) {
	fmt.Println("Monitoring alerts...")
	if lastAlertSentTime == nil {
		lastAlertSentTime = make(map[int32]time.Time)
	}
	for {
		msg := <-monitorChan

		if msg.Msg == "sys_stat" {
			fmt.Println("Sys stat received", string(msg.Data))
			var sysStat SystemStat
			err := sysStat.FromBytes(msg.Data)
			if err != nil {
				fmt.Println("Error decoding sys_stat", err)
				continue
			}
			go checkPercentUsage(ctx, repo, msg.NodeId, "cpu", "CPU", average(sysStat.CPUUsage))
			go checkPercentUsage(ctx, repo, msg.NodeId, "mem", "Memory", sysStat.MemUsage)
			go checkPercentUsage(ctx, repo, msg.NodeId, "disk", "Disk", sysStat.DiskUsage)
			go checkNetworkUsage(ctx, repo, msg.NodeId, float64(sysStat.NetSentPS), float64(sysStat.NetRecvPS))
		}
	}
}

func average(nums []float64) float64 {
	if len(nums) == 0 {
		return 0 // avoid division by zero
	}

	var sum float64
	for _, num := range nums {
		sum += num
	}

	return sum / float64(len(nums))
}

// sendAlertNotifications sends alert to all configured notification channels
func sendAlertNotifications(alert db.GetActiveAlertsByNodeAndMetricRow, alertMsg AlertMsg) {
	// Send Discord alert if webhook is configured
	if alert.DiscordWebhook.String != "" {
		if err := SendDiscordAlert(alert.DiscordWebhook.String, alertMsg); err != nil {
			fmt.Printf("Failed to send Discord alert: %v\n", err)
		}
	}

	// Send Email alert if email is configured
	if alert.Email.String != "" {
		if err := SendEmailAlert(alert.Email.String, alertMsg); err != nil {
			fmt.Printf("Failed to send email alert: %v\n", err)
		}
	}

	// Send Slack alert if webhook is configured
	if alert.SlackWebhook.String != "" {
		if err := SendSlackAlert(alert.SlackWebhook.String, alertMsg); err != nil {
			fmt.Printf("Failed to send Slack alert: %v\n", err)
		}
	}
}

// checkPercentUsage sends the alerts for a percentage metric (cpu, mem or
// disk) whose value is above their threshold. label names the metric in the
// notification.
func checkPercentUsage(ctx context.Context, repo *db.Repo, nodeId int32, metric string, label string, value float64) {
	alerts, err := repo.Queries.GetActiveAlertsByNodeAndMetric(ctx, db.GetActiveAlertsByNodeAndMetricParams{
		NodeID: int64(nodeId),
		Metric: metric,
	})
	if err != nil {
		fmt.Println("Error getting active alerts", err)
		return
	}
	if len(alerts) == 0 {
		fmt.Println("No active alerts found")
		return
	}
	fmt.Println("Active alerts found", len(alerts))
	for _, alert := range alerts {
		if value > alert.Threshold.Float64 {
			fmt.Println(label, "usage exceeded threshold for alert", int32(alert.ID))
			lastSendTime, ok := lastAlertSentTime[int32(alert.ID)]
			if ok {
				if time.Since(lastSendTime).Minutes() < float64(alert.Duration) {
					fmt.Println("Alert already sent within last", alert.Duration, "minutes")
					continue
				}
			}

			lastAlertSentTime[int32(alert.ID)] = time.Now()
			// send notifications to all configured channels
			sendAlertNotifications(alert, AlertMsg{
				NodeName:     alert.NodeName.String,
				NodeIp:       alert.NodeIp,
				Metric:       label,
				Threshold:    fmt.Sprintf("%.2f%%", alert.Threshold.Float64),
				CurrentValue: fmt.Sprintf("%.2f%%", value),
				Timestamp:    time.Now(),
			})
		}
	}
}

func checkNetworkUsage(ctx context.Context, repo *db.Repo, nodeId int32, netSend float64, netRecv float64) {
	alerts, err := repo.Queries.GetActiveAlertsByNodeAndMetric(ctx, db.GetActiveAlertsByNodeAndMetricParams{
		NodeID: int64(nodeId),
		Metric: "net",
	})
	if err != nil {
		fmt.Println("Error getting active alerts", err)
		return
	}
	if len(alerts) == 0 {
		fmt.Println("No active alerts found")
		return
	}
	fmt.Println("Active alerts found", len(alerts))
	for _, alert := range alerts {
		if netOverThreshold(netSend, netRecv, alert.NetSendThreshold.Float64, alert.NetReceThreshold.Float64) {
			fmt.Println("Network usage exceeded threshold for alert", int32(alert.ID))
			lastSendTime, ok := lastAlertSentTime[int32(alert.ID)]
			if ok {
				if time.Since(lastSendTime).Minutes() < float64(alert.Duration) {
					fmt.Println(" net Alert already sent within last", alert.Duration, "minutes")
					continue
				}
			}

			lastAlertSentTime[int32(alert.ID)] = time.Now()
			// send notifications to all configured channels
			sendAlertNotifications(alert, AlertMsg{
				NodeName:     alert.NodeName.String,
				NodeIp:       alert.NodeIp,
				Metric:       "Network",
				Threshold:    fmt.Sprintf("Send: %s, Recv: %s", netThresholdText(alert.NetSendThreshold.Float64), netThresholdText(alert.NetReceThreshold.Float64)),
				CurrentValue: fmt.Sprintf("Send: %s, Recv: %s", formatBytesPerSec(netSend), formatBytesPerSec(netRecv)),
				Timestamp:    time.Now(),
			})
		}
	}
}

// netOverThreshold reports whether sent or received bytes per second are
// above their own threshold. A threshold of 0 (left empty in the form) means
// that direction isn't checked.
func netOverThreshold(sent, recv, sendThreshold, recvThreshold float64) bool {
	return (sendThreshold > 0 && sent > sendThreshold) ||
		(recvThreshold > 0 && recv > recvThreshold)
}

// netThresholdText shows an unchecked (0) threshold as "off".
func netThresholdText(threshold float64) string {
	if threshold <= 0 {
		return "off"
	}
	return formatBytesPerSec(threshold)
}

// formatBytesPerSec uses decimal units (1 KB = 1000 B), like the dashboard.
func formatBytesPerSec(v float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	i := 0
	for v >= 1000 && i < len(units)-1 {
		v /= 1000
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", v, units[i])
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
