package controllers

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/akhil/go-fiber-postgres/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Auth struct {
	DB *gorm.DB
}

type LoginInput struct {
	Email *string `json:"email"` // ใช้ string pointer เพื่อความสะดวกในการเก็บค่า ใน db สามารถตรวจสอบค่าได้ง่าย
	//ถ้า pointer Email เป็น nil แต่เป็นstring ว่าง แสดงว่ามีการส่งมาแต่เป็นค่าว่าง แต่ pointer Email เป็น not nil แสดงว่าไม่มีการส่งมาด้วยซ้ำ
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
	group.Get("/me", db.GetUserInfo)
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
	cookie.Domain = "localhost"
	cookie.Name = "jwt"        // ชื่อ cookie
	cookie.Value = signedToken // ใส่ token ลงไป
	cookie.Expires = expTime   // กำหนดเวลาหมดอายุให้ตรงกับ exp claim
	cookie.HTTPOnly = false    // ป้องกัน JavaScript ฝั่ง client อ่านได้
	cookie.Secure = false      // สั่งให้ส่งเฉพาะบน HTTPS (production)
	cookie.SameSite = "Lax"    // หรือตั้งเป็น "Strict" / "None" ตามความเหมาะสม

	// เซ็ต cookie ลงใน response
	c.Cookie(cookie)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "login successful",
		"token":   signedToken,
		"exp":     expTime,
	})

}

type UserClaims struct { //โครงสร้างสำหรับ ดึงข้อมูลมาจากะนาำื
	UserID               uint    `json:"id"`
	Email                *string `json:"email"`
	Role                 *string `json:"role"`
	jwt.RegisteredClaims         //นำข้อมูลอื่นๆนอกจาก 3 ตัวบน มาด้วย แต่หลักๆจะใช้แค่ 3 คัว
}

// ฟังก์ชันสำหรับ แปลง Token ให้สามารถอ่านค่าจากภายในได้ เดวจะต้องนำไปใช้ตอนดึง token ออกมาจาก cookie
func VerifyJWT(tokenString string) (*UserClaims, error) { //รับค่า token ที่ได้จาก cookie
	var jwtSecret = []byte(os.Getenv("Jwt_Secret")) // แปลง Secret ให้อยู่ในรูปแบบรหัส
	ptr := &UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, ptr, func(tokenCheck *jwt.Token) (interface{}, error) {
		//รับค่า token และ model ที่มารองรับ payload และ ตรวจสอบ alhorithm ว่าถูกสร้างด้วยอันเดียวกันไหมถ้าถูกต้องจะเอา secret มาเป็น parameter นี้แทน
		//ถ้าทุกอย่างถูกหมด ตัวแปร token จะสามารถเข้าถึงข้อมูลใน token ได้(เป็น struct)
		if _, ok := tokenCheck.Method.(*jwt.SigningMethodHMAC); !ok { //เช็คว่า token ถูก เข้ารหัส ด้าย algo นี้รึป่าว ถ้าใช่จะคืนค่า function เป็น secret

			return nil, errors.New("unexpected signing method")
		}

		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	return ptr, nil
}

func (db *Auth) GetUserInfo(c *fiber.Ctx) error {
	// ดึง JWT จาก cookie ที่ชื่อ jwt
	tokenString := c.Cookies("jwt")
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "no token"})
	}

	// ตรวจสอบและ decode JWT
	claims, err := VerifyJWT(tokenString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// ส่งข้อมูล user ออกไป (แค่ตัวอย่าง user_id, email, role)
	return c.Status(200).JSON(fiber.Map{
		"user_id": claims.UserID,
		"email":   *claims.Email,
		"role":    *claims.Role,
		"token":   tokenString,
	})
}

func (db *Auth) Logout(c *fiber.Ctx) error {
	// ลบ cookie 'jwt' โดย set ค่าใหม่เป็นว่างและ expired ย้อนหลัง
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // set เวลาย้อนหลัง
		HTTPOnly: true,
		Secure:   false, // true เฉพาะถ้าเป็น https
		SameSite: "Lax",
	})
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}
