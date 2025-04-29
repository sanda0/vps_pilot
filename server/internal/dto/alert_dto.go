package dto

type AlertDto struct {
	NodeID           int32   `json:"node_id" binding:"required"`
	Metric           string  `json:"metric" binding:"required"`
	Threshold        float64 `json:"threshold"`
	NetReceThreshold float64 `json:"net_rece_threshold"`
	NetSendThreshold float64 `json:"net_send_threshold"`
	Duration         int32   `json:"duration"`
	Email            string  `json:"email"`
	Discord          string  `json:"discord"`
	Slack            string  `json:"slack"`
	Enabled          bool    `json:"enabled"`
}

type AlertUpdateDto struct {
	ID               int32   `json:"id" binding:"required"`
	NodeID           int32   `json:"node_id" binding:"required"`
	Metric           string  `json:"metric" binding:"required"`
	Threshold        float64 `json:"threshold"`
	NetReceThreshold float64 `json:"net_rece_threshold"`
	NetSendThreshold float64 `json:"net_send_threshold"`
	Duration         int32   `json:"duration"`
	Email            string  `json:"email"`
	Discord          string  `json:"discord"`
	Slack            string  `json:"slack"`
	Enabled          bool    `json:"enabled"`
}

// export const AlertSchema = z.object({
//   metric: z.enum(["CPU", "Memory", "Network"], { required_error: "Select a metric" }),
//   threshold: z.number().min(0).max(100, "Threshold must be between 0-100"),
//   duration: z.number().min(1, "Duration must be at least 1 minute"),
//   // email: z.string().email().optional(),
//   discord: z.string().url().optional(),
//   enabled: z.boolean(),
// });
