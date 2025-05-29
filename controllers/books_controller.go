package controllers

import (
	"net/http"
	"os"

	"github.com/akhil/go-fiber-postgres/middleware"
	"github.com/akhil/go-fiber-postgres/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// BooksController handles book-related routes
type BooksController struct {
	DB *gorm.DB
}

// NewBooksController creates a new BooksController
func NewBooksController(db *gorm.DB) *BooksController {
	return &BooksController{DB: db}
}

// RegisterRoutes registers all book routes under /api/books
func (bc *BooksController) RegisterRoutes(app *fiber.App) {
	secret := os.Getenv("Jwt_Secret")
	group := app.Group("/api/books")
	group.Post("/", bc.CreateBook)
	group.Delete("/:id", bc.DeleteBook)
	group.Get("/:id", bc.GetBookByID)
	group.Get("/", middleware.RoleAuthMiddleware(secret, "admin"), bc.GetBooks)
	group.Patch("/:id", bc.UpdateBook)
}

// , middleware.RoleCookieMiddleware(os.Getenv("Jwt_Secret"), "jwt", "admin")
// CreateBook creates a new book record
func (bc *BooksController) CreateBook(c *fiber.Ctx) error {
	book := models.Books{}
	if err := c.BodyParser(&book); err != nil {
		c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"message": "request failed"})
		return err
	}
	if err := bc.DB.Create(&book).Error; err != nil {
		c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "could not create book"})
		return err
	}
	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "book has been added",
		"data":    book,
	})
	return nil
}

// DeleteBook deletes a book by ID
func (bc *BooksController) DeleteBook(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).
			JSON(fiber.Map{"message": "id cannot be empty"})
	}

	// 1) อ่านข้อมูลเล่มนั้นก่อน เพราะต้องเอาข้อมูลจาก DB มาแสดง
	var book models.Books
	if err := bc.DB.First(&book, id).Error; err != nil {
		// ถ้าไม่เจอเล่มนี้ใน DB
		return c.Status(http.StatusNotFound).
			JSON(fiber.Map{"message": "book not found"})
	}

	// 2) สั่งลบ
	if err := bc.DB.Delete(&models.Books{}, id).Error; err != nil {
		return c.Status(http.StatusBadRequest).
			JSON(fiber.Map{"message": "could not delete book"})
	}

	// 3) ส่ง response พร้อมข้อมูลเล่มที่ถูกลบ
	return c.Status(http.StatusOK).
		JSON(fiber.Map{
			"message": "book deleted successfully",
			"data":    book,
		})
}

// GetBooks fetches all books
func (bc *BooksController) GetBooks(c *fiber.Ctx) error {
	var books []models.Books
	if err := bc.DB.Find(&books).Error; err != nil {
		c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "could not get books"})
		return err
	}
	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "books fetched successfully",
		"data":    books,
	})
	return nil
}

// GetBookByID fetches a single book by ID
func (bc *BooksController) GetBookByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "id cannot be empty"})
		return nil
	}
	book := models.Books{}
	if err := bc.DB.First(&book, id).Error; err != nil {
		c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "could not get the book"})
		return err
	}
	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "book fetched successfully",
		"data":    book,
	})
	return nil
}

func (bc *BooksController) UpdateBook(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).
			JSON(fiber.Map{"message": "id cannot be empty"})
	}

	// 1) อ่านเรคอร์ดก่อน
	var book models.Books
	if err := bc.DB.First(&book, id).Error; err != nil {
		return c.Status(http.StatusNotFound).
			JSON(fiber.Map{"message": "book not found"})
	}

	// 2) แปลง payload จาก client ให้มีจรงกับ input
	var input models.Books
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).
			JSON(fiber.Map{"message": "invalid input"})
	}

	// 3) อัปเดตฟิลด์ที่ต้องการ
	if input.Author != nil {
		book.Author = input.Author
	}
	if input.Title != nil {
		book.Title = input.Title
	}
	if input.Publisher != nil {
		book.Publisher = input.Publisher
	}

	// 4) บันทึกกลับ DB
	if err := bc.DB.Save(&book).Error; err != nil {
		return c.Status(http.StatusInternalServerError).
			JSON(fiber.Map{"message": "could not update book"})
	}

	// 5) ส่งกลับข้อมูลที่อัปเดต
	return c.Status(http.StatusOK).
		JSON(fiber.Map{
			"message": "book updated successfully",
			"data":    book,
		})
}
