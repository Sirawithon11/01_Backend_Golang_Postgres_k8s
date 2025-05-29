package storage

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres" // นำ connection string ไปเชื่อม
	"gorm.io/gorm"            //เป็น config สำหรับการเชื่อม postgres กับ connent string ของ postgres
)

type Config struct {
	Host     string
	Port     string
	Password string
	User     string
	DBName   string
	SSLMode  string
}

func NewConnection() (*gorm.DB, error) {
	err := godotenv.Load(".env") // รับตัวแปร env
	if err != nil {
		log.Fatal(err)
	}

	config := &Config{ //กำหนดตัวแปร env ใส่ struct
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASS"),
		User:     os.Getenv("DB_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   os.Getenv("DB_NAME"),
	}

	dsn := fmt.Sprintf( //เป็นการนำ ตัวแปร env จาก database config มาประกอบให้อยู่ในรูปแบบ connection string สำหรับการเชื่อมต่อ
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{}) // นำ connention string ที่เราประกอบ มาเชื่อมต่อกับ Database ของจริง ผ่าน GORM และ return ตัวแปรสำหรับอ้างอิงทุกการเชื่อมต่อไป Database คือ DB
	if err != nil {
		fmt.Println(" Warning from exception : connectDB Error")
		return db, err
	}
	return db, nil // db ที่สำเร็จจากการเชื่อม เป็นตัวกลางประเภท *gorm.DB สำหรับอ้างอิง ถึง DB จริง
}
