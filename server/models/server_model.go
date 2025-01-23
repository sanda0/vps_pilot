package models

import "gorm.io/gorm"

type Server struct {
	gorm.Model
	Name       string
	IP         string
	AuthMethod string
	Username   string
	Password   string
	PrivateKey string
}

type ServerStat struct {
	gorm.Model
	ServerID   uint
	CpuLoad    float64
	MemoryUsed float64
	DiskUsed   float64
}
