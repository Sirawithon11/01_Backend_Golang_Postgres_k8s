package models

type Books struct { // สร้าง struct เป็น model สำหรับอ้างอิงถึง table ใน db จริง  (เช่น Books จะอ้างอิงถึงตารางชื่อ books และ ชื่อจะต้องเป็น plural เสมอ)
	ID uint `gorm:"primary key;autoIncrement" json:"id"` // 'Gorm และ json ด้านหลังเป็นการกำหนดค่าตามจุดประสงค์ เช่น Gorm เป็น config สำหรับ DB
	// และ json เป็น config สำหรับการเปลี่ยนค่าไปมา ระหว่าง map และ JSON จากการติดต่อจาก client (backend กับ DB ไม่ได้ติดต่อโดยใช้ api หรือ Http req.) '
	Author    *string `json:"author"` // *string เป็นการกำหนด type ทั้ง ของ map และ การเติม *เป็นการบอกใน postgres ไปด้วยในตัว ว่าเป็น string เป็น null ได้
	Title     *string `json:"title"`
	Publisher *string `json:"publisher"`
}
