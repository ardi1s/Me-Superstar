// models 包定义所有数据库模型（表结构），以及数据库初始化和自动迁移逻辑。
package models

import (
	"fmt"
	"log"

	"agent-backend/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := config.AppConfig.Database.DSN()

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	fmt.Println("Database connected successfully.")
}

func AutoMigrate() {
	err := DB.AutoMigrate(
		&User{},
		&Account{},
		&Work{},
		&WorkDailyStats{},
	)
	if err != nil {
		log.Fatalf("failed to auto migrate: %v", err)
	}

	fmt.Println("Auto migration completed.")
}
