package main

import (
	"UserService/config" // Replace with your actual module path
	"UserService/handler"
	"UserService/model"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func main() {
	// This is a placeholder for the main function.
	// You can add your code here to implement the desired functionality.
	// For example, you might want to initialize a server, run a loop, or perform some calculations.

	app := fiber.New()
	config.ConnectDB()
	config.ConnectRedis()
	config.Taobanguser()
	config.Taobangtoken()
	config.ConnectFirebase()

	app.Patch("/refreshtoken", func(c *fiber.Ctx) error {
		type TokenInput struct {
			Id           uint   `json:"id"`
			Refreshtoken string `json:"refreshtoken"`
			Accesstoken  string `json:"accesstoken"`
		}
		var input TokenInput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
		}
		token, err := tokencheck(input.Id, input.Refreshtoken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		// Trả về token mới
		return c.JSON(fiber.Map{
			"accesstoken": token.Accesstoken,
		})

	})
	app.Post("/getme", func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Get("X-User-ID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		var user model.User
		result := config.DB.Where("id = ?", id).First(&user)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
		}
		// Trả về thông tin người dùng
		return c.JSON(fiber.Map{
			"name":        user.Name,
			"avatar":      user.Avatar,
			"background":  user.Background,
			"description": user.Description,
		})

	})
	app.Post("/getuserprofile", func(c *fiber.Ctx) error {
		type Idinput struct {
			Id string `json:"idstr"`
		}
		var input Idinput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
		}
		id, err := strconv.Atoi(input.Id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		var user model.User
		result := config.DB.Where("id = ?", id).First(&user)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
		}
		// Trả về thông tin người dùng
		return c.JSON(fiber.Map{
			"name":        user.Name,
			"avatar":      user.Avatar,
			"background":  user.Background,
			"description": user.Description,
		})

	})

	app.Post("/updateme", func(c *fiber.Ctx) error {
		type Updateinput struct {
			Name        *string `json:"name,omitempty"`
			Background  *string `json:"background,omitempty"`
			Avatar      *string `json:"avatar,omitempty"`
			Description *string `json:"description,omitempty"`
		}
		var input Updateinput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
		}
		id, err := strconv.Atoi(c.Get("X-User-ID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		var user model.User
		result := config.DB.Where("id = ?", id).First(&user)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
		}
		// Cập nhật thông tin người dùng
		if input.Name != nil {

			user.Name = *input.Name
		}
		if input.Background != nil {

			user.Background = *input.Background
		}

		if input.Avatar != nil {
			user.Avatar = *input.Avatar
		}
		if input.Description != nil {
			user.Description = *input.Description
		}
		if err := config.DB.Save(&user).Error; err != nil {

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update user"})
		}
		ctx := context.Background()

		// Cập nhật thông tin người dùng trong Firestore
		_, err = config.FirestoreClient.Collection("user").Doc(strconv.Itoa(int(user.Id))).Set(ctx, map[string]interface{}{
			"name":        user.Name,
			"avatar":      user.Avatar,
			"background":  user.Background,
			"description": user.Description,
		}, firestore.MergeAll)
		if err != nil {
			log.Printf("Failed to update Firestore user: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update Firestore user"})
		}
		log.Printf("User %d updated in Firestore", user.Id)
		// Cập nhật thông tin người dùng trong Redis
		// Trả về thông tin người dùng đã cập nhật
		return c.JSON(fiber.Map{
			"name":        user.Name,
			"avatar":      user.Avatar,
			"background":  user.Background,
			"description": user.Description,
		})
	})
	app.Post("/deviceme", func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Get("X-User-ID"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		var user model.User
		result := config.DB.Where("id = ?", id).First(&user)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
		}
		var tokens []model.Token
		result = config.DB.Where("user_id = ?", id).Find(&tokens)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No devices found for user"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
		}
		return c.JSON(fiber.Map{
			"tokens": tokens,
		})

	})

	app.Listen(":8082")
	// Note: Replace "your_module_path" with the actual module path of your project.

}
func tokencheck(inputid uint, inputrefreshtoken string) (*model.Token, error) {
	var token model.Token
	result := config.DB.Where("id = ? ", inputid).Where("refreshtoken = ?", inputrefreshtoken).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cannot find token")
		}
		return nil, result.Error
	}
	log.Println("Token found:", token.Accesstoken)
	newtoken, err := handler.GenerateJWT(inputid)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}
	err = config.DB.Model(&token).Update("accesstoken", newtoken).Error
	if err != nil {
		return nil, fmt.Errorf("failed update token: %v", err)
	}
	token.Accesstoken = newtoken
	log.Println("Token updated:", token.Accesstoken)
	return &token, nil

}
