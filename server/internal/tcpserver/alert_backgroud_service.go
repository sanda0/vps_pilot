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
			go checkCpuUsage(ctx, repo, msg.NodeId, average(sysStat.CPUUsage))
			go checkMemoryUsage(ctx, repo, msg.NodeId, sysStat.MemUsage)
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

func checkCpuUsage(ctx context.Context, repo *db.Repo, nodeId int32, cpuAvg float64) {
	alerts, err := repo.Queries.GetActiveAlertsByNodeAndMetric(ctx, db.GetActiveAlertsByNodeAndMetricParams{
		NodeID: nodeId,
		Metric: "cpu",
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
		if cpuAvg > alert.Threshold.Float64 {
			fmt.Println("Cpu usage exceeded threshold for alert", alert.ID)
			lastSendTime, ok := lastAlertSentTime[alert.ID]
			if ok {
				if time.Since(lastSendTime).Minutes() < float64(alert.Duration) {
					fmt.Println("Alert already sent within last", alert.Duration, "minutes")
					continue
				}
			}

			lastAlertSentTime[alert.ID] = time.Now()
			// send notifications to all configured channels
			sendAlertNotifications(alert, AlertMsg{
				NodeName:     alert.NodeName.String,
				NodeIp:       alert.NodeIp,
				Metric:       "CPU",
				Threshold:    fmt.Sprintf("%.2f%%", alert.Threshold.Float64),
				CurrentValue: fmt.Sprintf("%.2f%%", cpuAvg),
				Timestamp:    time.Now(),
			})
		}
	}
}

func checkMemoryUsage(ctx context.Context, repo *db.Repo, nodeId int32, memUsage float64) {
	alerts, err := repo.Queries.GetActiveAlertsByNodeAndMetric(ctx, db.GetActiveAlertsByNodeAndMetricParams{
		NodeID: nodeId,
		Metric: "mem",
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
		if memUsage > alert.Threshold.Float64 {
			fmt.Println("Memory usage exceeded threshold for alert", alert.ID)
			lastSendTime, ok := lastAlertSentTime[alert.ID]
			if ok {
				if time.Since(lastSendTime).Minutes() < float64(alert.Duration) {
					fmt.Println("Alert already sent within last", alert.Duration, "minutes")
					continue
				}
			}

			lastAlertSentTime[alert.ID] = time.Now()
			// send notifications to all configured channels
			sendAlertNotifications(alert, AlertMsg{
				NodeName:     alert.NodeName.String,
				NodeIp:       alert.NodeIp,
				Metric:       "Memory",
				Threshold:    fmt.Sprintf("%.2f%%", alert.Threshold.Float64),
				CurrentValue: fmt.Sprintf("%.2f%%", memUsage),
				Timestamp:    time.Now(),
			})
		}
	}
}

func checkNetworkUsage(ctx context.Context, repo *db.Repo, nodeId int32, netSend float64, netRecv float64) {
	alerts, err := repo.Queries.GetActiveAlertsByNodeAndMetric(ctx, db.GetActiveAlertsByNodeAndMetricParams{
		NodeID: nodeId,
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
		if netSend > alert.Threshold.Float64 || netRecv > alert.Threshold.Float64 {
			fmt.Println("Network usage exceeded threshold for alert", alert.ID)
			lastSendTime, ok := lastAlertSentTime[alert.ID]
			if ok {
				if time.Since(lastSendTime).Minutes() < float64(alert.Duration) {
					fmt.Println(" net Alert already sent within last", alert.Duration, "minutes")
					continue
				}
			}

			lastAlertSentTime[alert.ID] = time.Now()
			// send notifications to all configured channels
			sendAlertNotifications(alert, AlertMsg{
				NodeName:     alert.NodeName.String,
				NodeIp:       alert.NodeIp,
				Metric:       "Network",
				Threshold:    fmt.Sprintf("Send: %.2f%%, Recv: %.2f%%", alert.NetSendThreshold.Float64, alert.NetReceThreshold.Float64),
				CurrentValue: fmt.Sprintf("Send: %.2f%%, Recv: %.2f%%", netSend, netRecv),
				Timestamp:    time.Now(),
			})
		}
	}
}
