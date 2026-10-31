package dto

import "github.com/sanda0/vps_pilot/internal/tcpserver"

// LogFilesDto lists a project's readable log files. Error is set when the
// agent refused, or cut the list at 1000 files.
type LogFilesDto struct {
	Files []tcpserver.LogFile `json:"files"`
	Error string              `json:"error"`
}

// OpenLogRequest asks for up to MaxBytes ending at Before (nil: the end of
// the file), then, with Follow, what is appended after it.
type OpenLogRequest struct {
	File     string
	Before   *int64
	MaxBytes int64
	Follow   bool
}
