package dto

import (
	"encoding/json"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/utils"
)

type NodeWithSysInfoDto struct {
	ID              int32   `json:"id"`
	Name            string  `json:"name"`
	Ip              string  `json:"ip"`
	Os              string  `json:"os"`
	Platform        string  `json:"platform"`
	PlatformVersion string  `json:"platform_version"`
	KernelVersion   string  `json:"kernel_version"`
	Cpus            int32   `json:"cpus"`
	TotalMemory     float64 `json:"total_memory"`
}

func (n *NodeWithSysInfoDto) Convert(row *db.GetNodesWithSysInfoRow) {
	n.ID = row.ID
	n.Name = row.Name.String
	n.Ip = row.Ip
	n.Os = row.Os.String
	n.Platform = row.Platform.String
	n.PlatformVersion = row.PlatformVersion.String
	n.KernelVersion = row.KernelVersion.String
	n.Cpus = row.Cpus.Int32
	n.TotalMemory = utils.BytesToGB(row.TotalMemory.Float64)
}

type NodeNameUpdateDto struct {
	NodeId int32  `json:"id"`
	Name   string `json:"name"`
}

type NodeDto struct {
	ID     int32   `json:"id"`
	Name   string  `json:"name"`
	Ip     string  `json:"ip"`
	Memory float64 `json:"memory"`
	Cpus   int32   `json:"cpus"`
}

type SystemStatQueryDto struct {
	Node      db.Node `json:"node" `
	StatType  string  `json:"stat_type" binding:"required"`
	TimeRange string  `json:"time_range" `
}

type SystemStatResponseDto struct {
	NodeID    int32                    `json:"node_id"`
	TimeRange string                   `json:"time_range"`
	Cpu       []map[string]interface{} `json:"cpu"`
	Mem       []db.GetSystemStatsRow   `json:"mem"`
}

func (s *SystemStatResponseDto) ToBytes() ([]byte, error) {
	return json.Marshal(s)
}

type NodeSystemStatRequestDto struct {
	ID        int32  `json:"id"`
	TimeRange string `json:"time_range"`
}

func (n *NodeSystemStatRequestDto) FromBytes(data []byte) error {
	err := json.Unmarshal(data, n)
	if n.TimeRange == "5M" {
		n.TimeRange = "5 minutes"
	} else if n.TimeRange == "15M" {
		n.TimeRange = "15 minutes"
	} else if n.TimeRange == "1H" {
		n.TimeRange = "1 hour"
	} else if n.TimeRange == "1D" {
		n.TimeRange = "1 day"
	} else if n.TimeRange == "2D" {
		n.TimeRange = "2 days"
	} else if n.TimeRange == "1W" {
		n.TimeRange = "1 week"
	}
	return err
}
