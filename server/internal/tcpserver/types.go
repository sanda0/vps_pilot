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
	Errors   []ProjectError  `json:"errors"`
}

// ProjectError is a config the agent couldn't use, or a rejected manual path.
type ProjectError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// ProjectSettings tells the agent how to search for projects.
type ProjectSettings struct {
	AutoScan     bool     `json:"auto_scan"`
	ScanInterval int64    `json:"scan_interval"` // seconds
	Paths        []string `json:"paths"`
}

func (p *ProjectSyncPayload) FromBytes(data []byte) error {
	return json.Unmarshal(data, p)
}

type SystemInfo struct {
	Hostname        string   `json:"hostname"`         // used as the default node name
	OS              string   `json:"os"`               // e.g. linux, windows
	Platform        string   `json:"platform"`         // e.g. ubuntu, centos
	PlatformVersion string   `json:"platform_version"` // e.g. 20.04, 8
	KernelVersion   string   `json:"kernel_version"`   // e.g. 5.4.0-42-generic
	CPUs            int      `json:"cpus"`             // number of CPUs
	TotalMemory     uint64   `json:"total_memory"`     // total memory in bytes
	AgentVersion    string   `json:"agent_version"`    // empty before v0.3
	Features        []string `json:"features"`         // e.g. ["commands", "logs"]; none before v0.3
}

// Features an agent can list in SystemInfo (docs/PROTOCOL.md).
const (
	FeatureCommands = "commands" // handles command_run, since v0.3
	FeatureLogs     = "logs"     // handles log_list and log_open, since v0.4
)

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

// Msg is the frame envelope on the agent channel. See docs/PROTOCOL.md.
type Msg struct {
	Msg    string          `json:"type"`
	NodeId int32           `json:"node_id,omitempty"`
	Token  string          `json:"token,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// Message types on the agent channel.
const (
	MsgEnroll     = "enroll"      // agent → hub, Token = join secret, Data = SystemInfo
	MsgEnrolled   = "enrolled"    // hub → agent, Data = EnrolledPayload
	MsgConnected  = "connected"   // agent → hub, Token = "<agent_id>:<node_key>", Data = SystemInfo
	MsgAuthFailed = "auth_failed" // hub → agent, Data = AuthFailedPayload; connection is closed
	MsgSysStat    = "sys_stat"    // hub → agent: start streaming; agent → hub: Data = SystemStat
	MsgSysInfo    = "sys_info"    // agent → hub, Data = SystemInfo
	MsgProjects   = "projects"    // agent → hub, Data = ProjectSyncPayload

	MsgProjectSettings = "project_settings" // hub → agent, Data = ProjectSettings
	MsgRescanProjects  = "rescan_projects"  // hub → agent: search for projects now

	MsgCommandRun    = "command_run"    // hub → agent, Data = CommandRun
	MsgCommandCancel = "command_cancel" // hub → agent, Data = CommandCancel
	MsgCommandOutput = "command_output" // agent → hub, Data = CommandOutput
	MsgCommandExit   = "command_exit"   // agent → hub, Data = CommandExit

	MsgLogList  = "log_list"  // hub → agent, Data = LogList
	MsgLogFiles = "log_files" // agent → hub, Data = LogFiles
	MsgLogOpen  = "log_open"  // hub → agent, Data = LogOpen
	MsgLogClose = "log_close" // hub → agent, Data = LogClose
	MsgLogData  = "log_data"  // agent → hub, Data = LogData
	MsgLogEnd   = "log_end"   // agent → hub, Data = LogEnd
)

// CommandRun asks the agent to run a command by name. Never a command string.
type CommandRun struct {
	RunID       string `json:"run_id"`
	ProjectPath string `json:"project_path"`
	Command     string `json:"command"`
}

type CommandCancel struct {
	RunID string `json:"run_id"`
}

// CommandOutput is one chunk of a run's stdout or stderr.
type CommandOutput struct {
	RunID  string `json:"run_id"`
	Seq    int64  `json:"seq"`
	Stream string `json:"stream"` // stdout or stderr
	Data   string `json:"data"`
}

// CommandExit ends a run. Status is exited, timeout, cancelled, rejected or
// failed; the hub adds lost for runs whose agent disconnected.
type CommandExit struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	ExitCode   *int64 `json:"exit_code"`
	RunAs      string `json:"run_as"`
	DurationMs int64  `json:"duration_ms"`
	Truncated  bool   `json:"truncated"`
	Error      string `json:"error"`
}

// LogList asks for the files in a project's log folders.
type LogList struct {
	RequestID   string `json:"request_id"`
	ProjectPath string `json:"project_path"`
}

// LogFile is a readable file; Path is relative to the project directory.
type LogFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"` // Unix seconds
}

// LogFiles answers a LogList. Error is set when the request was refused or
// the list was cut short.
type LogFiles struct {
	RequestID string    `json:"request_id"`
	Files     []LogFile `json:"files"`
	Error     string    `json:"error"`
}

// LogOpen asks for up to MaxBytes ending at Before (nil: the end of the
// file), then, with Follow, what is appended after it.
type LogOpen struct {
	StreamID    string `json:"stream_id"`
	ProjectPath string `json:"project_path"`
	File        string `json:"file"`
	Before      *int64 `json:"before"`
	MaxBytes    int64  `json:"max_bytes"`
	Follow      bool   `json:"follow"`
}

type LogClose struct {
	StreamID string `json:"stream_id"`
}

// LogData is one chunk of a stream. Offset is where Data starts in the file
// and Size is the file's size when it was read.
type LogData struct {
	StreamID string `json:"stream_id"`
	Seq      int64  `json:"seq"`
	Offset   int64  `json:"offset"`
	Data     string `json:"data"`
	Size     int64  `json:"size"`
	Rotated  bool   `json:"rotated"`
}

// LogEnd ends a stream. Reason is done, closed, rejected, error or limit.
type LogEnd struct {
	StreamID string `json:"stream_id"`
	Reason   string `json:"reason"`
	Error    string `json:"error"`
}

// EnrolledPayload carries the long-lived credentials issued at enrollment.
type EnrolledPayload struct {
	AgentID string `json:"agent_id"`
	NodeKey string `json:"node_key"`
	NodeID  int64  `json:"node_id"`
}

// AuthFailedPayload tells the agent why it was rejected.
type AuthFailedPayload struct {
	Reason string `json:"reason"`
}
