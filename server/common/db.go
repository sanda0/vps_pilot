package common

import (
	"os"

	"github.com/sanda0/vps_pilot/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Conn struct {
	Db *gorm.DB
}

func (c *Conn) Connect() *gorm.DB {
	if c.Db != nil {
		return c.Db
	} else {
		dburl := os.Getenv("DB_URL")
		var err error
		c.Db, err = gorm.Open(mysql.Open(dburl), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
		return c.Db
	}
}

func (c *Conn) Close() {
	if c.Db != nil {
		db, _ := c.Db.DB()
		db.Close()
	}
}

func (c *Conn) Migrate() {
	db := c.Connect()
	db.AutoMigrate(
		&models.User{},
		&models.Server{},
		&models.ServerStat{},
	)

}
