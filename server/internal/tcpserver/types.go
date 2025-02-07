package tcpserver

import "encoding/json"

type SystemInfo struct {
	OS              string `json:"os"`               // e.g. linux, windows
	Platform        string `json:"platform"`         // e.g. ubuntu, centos
	PlatformVersion string `json:"platform_version"` // e.g. 20.04, 8
	KernelVersion   string `json:"kernel_version"`   // e.g. 5.4.0-42-generic
	CPUs            int    `json:"cpus"`             // number of CPUs
	TotalMemory     uint64 `json:"total_memory"`     // total memory in bytes
}

func (s *SystemInfo) FromBytes(data []byte) error {
	return json.Unmarshal(data, s)
}

type Disk struct {
	Device     string `json:"device"`     // e.g. /dev/sda1
	Mountpoint string `json:"mountpoint"` // e.g. /
	Fstype     string `json:"fstype"`     // e.g. ext4
	Opts       string `json:"opts"`       // e.g. rw
	Total      uint64 `json:"total"`      // total disk space in bytes
	Used       uint64 `json:"used"`       // used disk space in bytes
}

type SystemStat struct {
	CPUUsage  []float64 `json:"cpu_usage"`
	MemUsage  float64   `json:"mem_usage"`
	DiskUsage float64   `json:"disk_usage"`
}

func (s *SystemStat) FromBytes(data []byte) error {
	return json.Unmarshal(data, s)
}

type Msg struct {
	Msg    string
	NodeId int32
	Token  string
	Data   []byte
}
