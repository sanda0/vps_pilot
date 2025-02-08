// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package db

import (
	"database/sql"
	"time"
)

type Node struct {
	ID        int32          `json:"id"`
	Name      sql.NullString `json:"name"`
	Ip        string         `json:"ip"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type NodeDiskInfo struct {
	ID         int32           `json:"id"`
	NodeID     int32           `json:"node_id"`
	Device     sql.NullString  `json:"device"`
	MountPoint sql.NullString  `json:"mount_point"`
	Fstype     sql.NullString  `json:"fstype"`
	Total      sql.NullFloat64 `json:"total"`
	Used       sql.NullFloat64 `json:"used"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type NodeSysInfo struct {
	ID              int32           `json:"id"`
	NodeID          int32           `json:"node_id"`
	Os              sql.NullString  `json:"os"`
	Platform        sql.NullString  `json:"platform"`
	PlatformVersion sql.NullString  `json:"platform_version"`
	KernelVersion   sql.NullString  `json:"kernel_version"`
	Cpus            sql.NullInt32   `json:"cpus"`
	TotalMemory     sql.NullFloat64 `json:"total_memory"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type SystemStat struct {
	Time     time.Time `json:"time"`
	NodeID   int32     `json:"node_id"`
	StatType string    `json:"stat_type"`
	CpuID    int32     `json:"cpu_id"`
	Value    float64   `json:"value"`
}

type User struct {
	ID           int32        `json:"id"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	PasswordHash string       `json:"password_hash"`
	CreatedAt    sql.NullTime `json:"created_at"`
	UpdatedAt    sql.NullTime `json:"updated_at"`
}
