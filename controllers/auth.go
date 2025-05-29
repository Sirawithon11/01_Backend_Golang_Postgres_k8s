package controllers

import (
	"net/http"
	"os"
	"time"

	"github.com/akhil/go-fiber-postgres/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Auth struct {
	DB *gorm.DB
}

type LoginInput struct {
	Email *string `json:"email"` // ใช้ string pointer เพื่อความสะดวกในการเก็บค่า ใน db สามารถตรวจสอบค่าได้ง่าย
	//เช่น ถ้า pointer Email เป็น nil แต่เป็นstring ว่าง แสดงว่ามีการส่งมาแต่เป็นค่าว่าง แต่ pointer Email เป็น not nil แสดงว่าไม่มีการส่งมาด้วยซ้ำ
	//และถ้า Pointer ที่นำไปใช้ ใน Bodyparser เป็น nil เราก็ไม่สามารถ เช็คได้ว่าลืมส่งอะไร
	Password *string `json:"password"`
}

// Gen JWT และเก็บข้อมูลที่ต้องการของผู้ login ใส่ payload
func GenerateJWT(user models.Users) (string, time.Time, error) {

	isNow := time.Now()                                // ตรวจสอบ เวลาปัจจุบัน
	exp := isNow.Add(24 * time.Hour)                   // กำหนดเวลาปัจจุบัน + เวลาที่จะเพิ่มเข้าไป
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, // ระบุอัลกอริธึม สำหรับการเข้ารหัส token
		jwt.MapClaims{ // ใส่ payload (ข้อมูลของ user ที่ login)
			"id":    user.Id,
			"email": user.Email,
			"role":  user.Role,
			"iat":   isNow.Unix(), //เวลาที่มีการสร้าง token
			"exp":   exp.Unix(),   //กำหนดเวลาหมดอายุของ token โดยการนำเวลา ปัจจุบัน เพิ่มระเวลาตามที่เรากำหนด
		})

	signedToken, err := token.SignedString([]byte(os.Getenv("Jwt_Secret"))) //secret ที่เรากำหนดขึ้นมาเอง ใช่ร่วมกับ algorithm เพื่อป้องกันการปลอมแปลง
	if err != nil {
		return "", exp, err
	}

	return signedToken, exp, nil
}

func DbForAuth(db *gorm.DB) *Auth {
	return &Auth{DB: db}
}

func (db *Auth) RegisterRoutes(app *fiber.App) {
	group := app.Group("/api/auth")
	group.Post("/register", db.registerUser)
	group.Post("/login", db.login)
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10) // 10 คือความซับซ้อนของการเข้ารหัส
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (db *Auth) registerUser(c *fiber.Ctx) error {
	user := &models.Users{}

	//นำข้อมูลออกมา จาก req.body
	if err := c.BodyParser(user); err != nil { // เป็นการ return nail จาก method ที่คืนค่า data type error
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"message": "register failed"})
	}

	if user.Email == nil || *user.Email == "" { // user และ email เป็น pointer ทั้งคู่  เช็คpointer ว่าได้ชี้ไหม หรือ หรือค่าที่ pointer ชี้ เป็นค่าว่างไหม
		return c.Status(http.StatusUnprocessableEntity).
			JSON(fiber.Map{"message": "email is required"})
	}
	if user.Password == nil || *user.Password == "" { // user และ email เป็น pointer ทั้งคู่  เช็คpointer ว่าได้ชี้ไหม หรือ หรือค่าที่ pointer ชี้ เป็นค่าว่างไหม
		return c.Status(http.StatusUnprocessableEntity).
			JSON(fiber.Map{"message": "password is required"})
	}
	if user.Role == nil || *user.Role == "" { // user และ email เป็น pointer ทั้งคู่  เช็คpointer ว่าได้ชี้ไหม หรือ หรือค่าที่ pointer ชี้ เป็นค่าว่างไหม
		return c.Status(http.StatusUnprocessableEntity).
			JSON(fiber.Map{"message": "role is required"})
	}

	hashed, err := HashPassword(*user.Password)
	if err != nil {
		return c.Status(http.StatusInternalServerError).
			JSON(fiber.Map{"message": "could not hash password"})
	}
	*user.Password = hashed

	if err := db.DB.Create(user).Error; err != nil {
		c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "could not register"})
		return err
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "register successfully",
		"data":    *user,
	})

}

// nil ถ้าใช้กับตัวแปร หรือ pointer หมายถึง ไม่มีค่า หรือ ไม่ได้ชี้ไปไหน
// แต่ การ return nil ในพารามิเตอร์ชนิด error หมายถึง “ไม่มีข้อผิดพลาดใดๆ เกิดขึ้น” หรือ “สำเร็จ (success)”

func (db *Auth) login(c *fiber.Ctx) error {
	checkUser := &LoginInput{}

	if err := c.BodyParser(checkUser); err != nil {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"message": "login failed"})
	}
	//ตรวจสอบว่า Email และ password ได้กรอก มาไหม หรือ มาเป็นค่าว่าง
	if checkUser.Email == nil || *checkUser.Email == "" {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"message": "email is required"})
	} else if checkUser.Password == nil || *checkUser.Password == "" {
		return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"message": "password is required"})
	}
	// หา Email ใน postgres
	var user models.Users
	if err := db.DB.Where("email = ?", *checkUser.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "invalid credentials"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "server error"})
	}
	//นำ password จาก client และ จาก DB ที่ผ่านการ bcrypt มาเปรียบเทียบกันว่าเหมือนกันไหม
	if err := bcrypt.CompareHashAndPassword(
		[]byte(*user.Password),
		[]byte(*checkUser.Password),
	); err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "invalid credentials"})
	}

	signedToken, expTime, err := GenerateJWT(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "cannot generate token"})
	}

	// สร้าง cookie
	cookie := new(fiber.Cookie)
	cookie.Name = "jwt"        // ชื่อ cookie
	cookie.Value = signedToken // ใส่ token ลงไป
	cookie.Expires = expTime   // กำหนดเวลาหมดอายุให้ตรงกับ exp claim
	cookie.HTTPOnly = true     // ป้องกัน JavaScript ฝั่ง client อ่านได้
	cookie.Secure = true       // สั่งให้ส่งเฉพาะบน HTTPS (production)
	cookie.SameSite = "Lax"    // หรือตั้งเป็น "Strict" / "None" ตามความเหมาะสม

	// เซ็ต cookie ลงใน response
	c.Cookie(cookie)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "login successful",
		"token":   signedToken,
		"exp":     expTime,
	})

}
