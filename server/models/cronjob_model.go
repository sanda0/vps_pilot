package models

import "gorm.io/gorm"

type CronJob struct {
	gorm.Model
	ServerID    uint
	Name        string
	Description string
	Schedule    string
	Commands    []string
}

type CronJobLog struct {
	gorm.Model
	CronJobID uint
	Command   string
	Output    string
}
