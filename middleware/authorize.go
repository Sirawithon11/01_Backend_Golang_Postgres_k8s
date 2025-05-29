package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
)

type UserClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

func RoleAuthMiddleware(secret string, allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).SendString("Missing or invalid Authorization header")
		}
		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid Token")
		}

		claims, ok := token.Claims.(*UserClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid Token Claims")
		}

		// ตรวจสอบ role
		for _, role := range allowedRoles {
			if claims.Role == role {
				c.Locals("user", claims)
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).SendString("Forbidden: insufficient role")
	}
}

//จากสิ่งเราทำ คือ login สร้าง token ส่ง token ไปทั้งทาง cookie และ res.body (จริงๆควรเลือกทางใดทางหนึ่งข้อดีข้อเสียต่างกัน) แต่ตอนยืนยันสิทธิเราใช้วิธี Http header (bearer token) เพื่อเข้าถึงข้อมูล ใน token (นำข้อมูล role มาใช้เข้าถึง)
//อีกวิธีหนึ่งคือ token ทีเก็บใน cookie ตลอดการส่ง api ไปมา จะมี token อยู่ใน cookie อยู่แล้ว สามารถเข้าถึงภายใน cookie เพื่อเอา toekn มาเข้ารหัส และเข้าถึง role หรือ ข้อมูลของผู้ใช้ ภายในได้
