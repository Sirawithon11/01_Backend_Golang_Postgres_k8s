package models

type Users struct {
	Id       uint    `gorm:"primaryKey; autoIncrement" json:"id"`
	Email    *string `gorm:"unique"  json:"email"` //Email เป็น pointer ที่ชี้ไปหา ตัวแปร String
	Password *string `json:"password"`
	Role     *string `json:"role"`
}
