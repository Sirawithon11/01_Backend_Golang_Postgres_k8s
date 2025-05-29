package main

import (
	"log"
	"os"

	"github.com/akhil/go-fiber-postgres/controllers"
	"github.com/akhil/go-fiber-postgres/models"
	"github.com/akhil/go-fiber-postgres/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	// นำ .env จากภายนอก พร้อมทำ exception
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}

	// เชื่อมต่อฐานข้อมูล
	db, err := storage.NewConnection() //db คือ object สำหรับอ้างอิงถือ postgres DB ที่ยอมรับการเชื่อมต่อแล้ว
	if err != nil {
		log.Fatal("could not connect to database:", err)
	}

	// รัน migration
	if err := models.MigrateBooks(db); err != nil {
		log.Fatal("could not migrate database:", err)
	}

	// เริ่มต้น Fiber app
	app := fiber.New()

	// ลงทะเบียน Routes
	booksController := controllers.NewBooksController(db) //ส่ง db object เข้าไปใน books_controller.go
	booksController.RegisterRoutes(app)                   // ส่ง server object(app จาก fiber new) เข้าไปสำหรับ ทำการ route

	// เริ่มเซิร์ฟเวอร์
	log.Fatal(app.Listen(":" + os.Getenv("Port")))
}
