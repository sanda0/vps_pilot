package dto

import "encoding/json"

// ProjectScanSettingsRequest is the body of PUT /nodes/:id/project-settings.
type ProjectScanSettingsRequest struct {
	AutoScan     *bool `json:"auto_scan" binding:"required"`
	ScanInterval int64 `json:"scan_interval" binding:"required,min=60,max=86400"` // seconds
}

// ProjectPathDto is a project directory added by hand.
type ProjectPathDto struct {
	ID        int64  `json:"id"`
	Path      string `json:"path"`
	CreatedAt int64  `json:"created_at"`
}

// ProjectScanDto is a node's project search settings and its last scan.
type ProjectScanDto struct {
	NodeID        int64            `json:"node_id"`
	AutoScan      bool             `json:"auto_scan"`
	ScanInterval  int64            `json:"scan_interval"`
	LastScannedAt *int64           `json:"last_scanned_at"` // unix seconds; null until the agent first reports
	Errors        json.RawMessage  `json:"errors"`          // [{"path": "...", "error": "..."}] from the last scan
	Paths         []ProjectPathDto `json:"paths"`
	Online        bool             `json:"online"` // the agent is connected, so changes and rescans apply now
}
