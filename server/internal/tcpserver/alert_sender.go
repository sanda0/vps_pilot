package tcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AlertMsg struct {
	NodeName     string
	NodeIp       string
	Metric       string
	Threshold    string
	CurrentValue string
	Timestamp    time.Time
}

func SendDiscordAlert(webhookURL string, alert AlertMsg) error {
	// Format the alert message
	message := fmt.Sprintf(
		"🚨 **ALERT** 🚨\nNode: `%s`\nIP: `%s`\nMetric: **%s**\nCurrent Value: `%s`\nThreshold: `%s`\nTimestamp: %s",
		alert.NodeName, alert.NodeIp, alert.Metric, alert.CurrentValue, alert.Threshold, alert.Timestamp.Format(time.RFC1123),
	)

	// Prepare the payload
	payload := map[string]string{
		"content": message,
	}

	// Marshal the payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send the POST request to the Discord webhook
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Alert sent to Discord!")
	} else {
		return fmt.Errorf("failed to send alert, status code: %d", resp.StatusCode)
	}

	return nil
}
