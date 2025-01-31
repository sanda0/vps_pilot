package common

import (
	"os"

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
