package controllers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/akhil/go-fiber-postgres/controllers"
	"github.com/akhil/go-fiber-postgres/models"
)

func ptr(s string) *string {
	return &s
}

// ช่วยตั้งค่า sqlmock + gorm.DB
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	// เปิด GORM ด้วย driver postgres ห่อ sql.DB
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true, // ลดการใช้ prepared statements
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	return gormDB, mock
}

func TestRegisterUser_Success(t *testing.T) {

	t.Run("add user successfully", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		auth := controllers.Auth{DB: gormDB}
		// เป็นคำสั่งเริ่มต้น สำหรับการ mock การทำงานของฐานข้อมูล
		mock.ExpectBegin()

		// 2) เป็นการสั่ง test mock การทำงานของฐานข้อมูล (จะมีกี่ test query ก็ได้ แต่ต้องอยู่ระหว่าง mock.Begin() และ mock.Commit())
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "users" ("email","password","role") VALUES ($1,$2,$3) RETURNING "id"`, //returning "id" คือการบอกว่าเราต้องการให้ฐานข้อมูลส่งค่า id กลับมาเมื่อมีการ insert ข้อมูลใหม่
		)).
			WithArgs(
				"john.doe@example.com", // email
				sqlmock.AnyArg(),       // hashed password (sqlmock.AnyArg() คำสั่งนี้คือรับค่าใดๆ ที่ไม่สนใจว่าเป็นอะไร เพราะเราจะไม่ตรวจสอบค่าของ password ในที่นี้ เพราะจริงๆแล้ว bcrypt จะทำการ hash password ให้เป็นค่าใหม่ทุกครั้งที่ไม่ใช่ String)
				"user",                 // role
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1)) //แสดงจำนวนแถวที่เกิดขึ้นจากการทำ query นี้(ที่มี column ชื่อ id) และ แถวนั้น id ต้องมีค่า 1

		// 3) เป็นคำสั่งสิ้นสุด สำหรับการ mock การทำงานของฐานข้อมูล
		mock.ExpectCommit()

		// json string สำหรับการทดสอบ
		payload := `{"email":"john.doe@example.com","password":"password123","role":"user"}`
		app := fiber.New()
		app.Post("/register", auth.RegisterUser)

		req := httptest.NewRequest("POST", "/register", strings.NewReader(payload)) //แปลง paylaod เป็น io.Reader สำหรับส่งไปกับ request
		req.Header.Set("Content-Type", "application/json")

		// เริ่มส่ง http.Request จำลองเข้าไปให้ Fiber ประมวลผลในหน่วยความจำ  //นำผมลัพธ์จาก api มาเก็บใน resp
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body) // resp.body คือ ข้อมูลที่มาจากการ response
		assert.NoError(t, err)

		var respBody struct {
			Message string       `json:"message"`
			Data    models.Users `json:"data"`
		}
		assert.NoError(t, json.Unmarshal(bodyBytes, &respBody)) //แปลง Body byte into struct ในของ ตัวแปร schema response

		assert.Equal(t, "register successfully", respBody.Message)
		assert.Equal(t, "john.doe@example.com", *respBody.Data.Email)
		assert.Equal(t, "user", *respBody.Data.Role)
		assert.NotEqual(t, "password123", *respBody.Data.Password) // ต้องถูก hash

		// 6) ยืนยันว่า mock expectations ถูกเรียกครบ
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("register user with missing value", func(t *testing.T) {
		app := fiber.New()
		app.Post("/register", controllers.Auth{}.RegisterUser)
		tests := []struct {
			description  string
			requestBody  models.Users
			expectStatus int
		}{
			{
				description: "empty email",
				requestBody: models.Users{
					Email:    ptr(""), // <-- ใช้ helper สร้าง *string
					Password: ptr("password123"),
					Role:     ptr("user"),
				},
				expectStatus: http.StatusUnprocessableEntity,
			},
			{
				description: "empty password",
				requestBody: models.Users{
					Email:    ptr("si@gmail.com"),
					Password: ptr(""), // <-- password ว่าง
					Role:     ptr("user"),
				},
				expectStatus: http.StatusUnprocessableEntity,
			},
			{
				description: "empty role",
				requestBody: models.Users{
					Email:    ptr("Si@gmail.com"),
					Password: ptr("password123"),
					Role:     ptr(""), // <-- role ว่าง
				},
				expectStatus: http.StatusUnprocessableEntity,
			},
		}

		// Run tests
		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				reqBody, _ := json.Marshal(test.requestBody)
				req := httptest.NewRequest("POST", "/users", bytes.NewReader(reqBody))
				req.Header.Set("Content-Type", "application/json")
				resp, _ := app.Test(req)

				assert.Equal(t, test.expectStatus, resp.StatusCode)
			})
		}
	})
}
