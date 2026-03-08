package tcpserver

import "encoding/json"

// ProjectCommand mirrors config.vpspilot.json commands entry
type ProjectCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// ProjectBackupDatabase holds DB connection env var names for backups
type ProjectBackupDatabase struct {
	Connection   string `json:"connection"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
}

// ProjectBackup holds the full backup configuration
type ProjectBackup struct {
	EnvFile     string                 `json:"env_file,omitempty"`
	ZipFileName string                 `json:"zip_file_name,omitempty"`
	Database    *ProjectBackupDatabase `json:"database,omitempty"`
	Dirs        []string               `json:"dir,omitempty"`
}

// ProjectConfig is the in-memory representation of a single config.vpspilot.json
type ProjectConfig struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"` // absolute path on the node where the config was found
	Tech     []string         `json:"tech"`
	Commands []ProjectCommand `json:"commands"`
	Logs     []string         `json:"logs"`
	Backups  *ProjectBackup   `json:"backups"`
}

// ProjectSyncPayload is what the agent sends inside Msg.Data when Msg.Msg == "projects"
// It contains the full list of projects currently found on the node so the server
// can upsert all of them in one shot.
type ProjectSyncPayload struct {
	Projects []ProjectConfig `json:"projects"`
}

func (p *ProjectSyncPayload) FromBytes(data []byte) error {
	return json.Unmarshal(data, p)
}

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
	NetSentPS int64     `json:"net_sent_ps"`
	NetRecvPS int64     `json:"net_recv_ps"`
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
