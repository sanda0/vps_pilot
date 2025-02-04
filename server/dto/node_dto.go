package dto

import (
	"github.com/sanda0/vps_pilot/db"
	"github.com/sanda0/vps_pilot/utils"
)

// type GetNodesWithSysInfoRow struct {
// 	ID              int32           `json:"id"`
// 	Name            sql.NullString  `json:"name"`
// 	Ip              string          `json:"ip"`
// 	Os              sql.NullString  `json:"os"`
// 	Platform        sql.NullString  `json:"platform"`
// 	PlatformVersion sql.NullString  `json:"platform_version"`
// 	KernelVersion   sql.NullString  `json:"kernel_version"`
// 	Cpus            sql.NullInt32   `json:"cpus"`
// 	TotalMemory     sql.NullFloat64 `json:"total_memory"`
// }

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
