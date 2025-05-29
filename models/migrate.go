package models

import "gorm.io/gorm" // ทุกครั้งที่มีการใช้ db หรือ ตัวแปรที่เป็นตัวกลางอ้างอิงไปถึง DB จริง(*gorm.DB) ต้องเรียกใช้ เสมอ

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&Books{},
		&Users{},
	)
	return err
}
