package main

import (
	"Yahoo/config"
	"crypto/rsa"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
)

func main() {

	app := fiber.New()
	config.ConnectRedis()

	// //================================================
	// tokenStr := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTExNzE2NTQsImlhdCI6MTc1MTE2ODA1NCwic3ViIjoiMyJ9.BsH3XXD2k-Z3Wwr9LJ6rbWyIw8NSnedqX7dzqpVsacuTypgPq-kdD0cM8UxcXYVPyHv0T8qtkANwh8uD1Tetbm0rVTKr_6Y_yGXVrFi0As1qR-kRlI8PSg9jujy2jcKrRGWhRAegk7HxPfUUPAWlyq3o6F8xpBPFtZAovdtG0wMCQK-JeGU7N4jSKcf4KEjTES7hmk4kG-dte9pbyWL_2fEpwlA_S9NG12nOdghg-geyif-NhSWCEc-ihnrTJoUOoGucWsM_agWPDLUIK3jA6LeuTqt1mSmhgkU6a6mP0gJOw1jJYUws3OZw2Gc1QIruOfz7eNb5adNKMENvJwZGLQ" // token nhận từ client

	// claims, err := VerifyJWT(tokenStr)
	// if err != nil {
	// 	log.Println("Token không hợp lệ:", err)

	// }
	// log.Println("Token hợp lệ. Claims:", (*claims)["sub"], (*claims)["exp"])
	// //=================================

	app.All("/auth/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		target := "http://localhost:8081/" + path
		log.Println("Proxy to:", target)
		log.Println()
		return proxy.Do(c, target)
	})
	app.Patch("/refreshtoken", func(c *fiber.Ctx) error {
		target := "http://localhost:8082/refreshtoken"
		log.Println("Proxy to:", target)
		log.Println()
		return proxy.Do(c, target)
	})
	app.Get("/getotp", func(c *fiber.Ctx) error {
		target := "http://localhost:8081/getotp"
		log.Println("Proxy to:", target)
		log.Println()
		return proxy.Do(c, target)
	})

	app.All("/user/*", UserMiddleware, func(c *fiber.Ctx) error {
		path := c.Params("*")
		claims := c.Locals("claims").(*jwt.MapClaims)
		userIdFloat, ok := (*claims)["sub"].(float64)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Invalid user ID in token",
			})
		}
		userId := strconv.Itoa(int(userIdFloat))
		capnhatonline(userId) // Cập nhật trạng thái online
		c.Request().Header.Set("X-User-ID", userId)

		target := "http://localhost:8082/" + path
		log.Println("Proxy to:", target)
		return proxy.Do(c, target)
	})
	app.Get("/getonline", func(c *fiber.Ctx) error {
		Id := c.Query("userid")
		if Id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Id is required",
			})
		}
		// Kiểm tra xem id có phải là số nguyên không
		if id, err := strconv.Atoi(Id); err != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Id must be a + number",
			})
		}
		// Lấy trạng thái online từ Redis
		redisKey := "online:" + Id

		val, err := config.Redis.Get(config.Ctx, redisKey).Result()
		if err == redis.Nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"online": false,
			})
		} else if err != nil {
			log.Println("Error getting online status:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		// Trả về trạng thái online
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"online": val == "1",
		})
	})

	app.Listen(":8080")

}

func loadPublicKey() (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile("../genkey/public.pem")
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPublicKeyFromPEM(keyData)
}
func VerifyJWT(tokenStr string) (*jwt.MapClaims, error) {
	publicKey, _ := loadPublicKey()
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Đảm bảo thuật toán đúng
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	// Nếu token hợp lệ và claims đúng định dạng
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func VerifyJWTIgnoreExp(tokenStr string) (*jwt.MapClaims, error) {
	publicKey, _ := loadPublicKey()

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		// Chỉ xác thực chữ ký
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	// KHÔNG kiểm tra token.Valid => không check hạn
	return &claims, nil
}

func UserMiddleware(c *fiber.Ctx) error {
	log.Println("UserMiddleware called")
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	// Format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization header format",
		})
	}
	tokenStr := parts[1]

	claims, err := VerifyJWT(tokenStr)
	if err != nil {
		log.Println("JWT error:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}
	// Gắn thông tin claims vào context để route sau dùng được

	jti, ok := (*claims)["jti"].(string)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot find token jti",
		})
	}
	isblacktoken, _ := IsTokenBlacklisted(jti)

	if isblacktoken {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User is disabled",
		})
	}
	role, ok := (*claims)["role"].(float64)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot find user role ",
		})
	}
	if ok && role == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User is disabled",
		})
	}
	// c.Request().Header.Set("User-Id", fmt.Sprintf("%v", (*claims)["sub"]))

	c.Locals("claims", claims)

	return c.Next()
}
func IsTokenBlacklisted(jti string) (bool, error) {
	key := "blacklist:" + jti
	val, err := config.Redis.Get(config.Ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

func AdminMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	// Format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization header format",
		})
	}
	tokenStr := parts[1]

	claims, err := VerifyJWT(tokenStr)
	if err != nil {
		log.Println("JWT error:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	jti, ok := (*claims)["jti"].(string)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot find token jti",
		})
	}
	isblacktoken, _ := IsTokenBlacklisted(jti)

	if isblacktoken {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User is disabled",
		})
	}
	// Gắn thông tin claims vào context để route sau dùng được
	role, ok := (*claims)["role"].(float64)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot find user role ",
		})
	}
	if ok && role < 2 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User is disabled",
		})
	}
	c.Locals("claims", claims)

	return c.Next()
}

func RefreshMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}
	// Format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization header format",
		})
	}
	tokenStr := parts[1]

	claims, err := VerifyJWTIgnoreExp(tokenStr)
	if err != nil {
		log.Println("JWT error:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	jti, ok := (*claims)["jti"].(string)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot find token jti",
		})
	}
	isblacktoken, _ := IsTokenBlacklisted(jti)

	if isblacktoken {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User is disabled",
		})
	}
	// Gắn thông tin claims vào context để route sau dùng được
	role, ok := (*claims)["role"].(float64)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot find user role ",
		})
	}
	if ok && role < 2 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User is disabled",
		})
	}
	c.Locals("claims", claims)

	return c.Next()
}
func capnhatonline(id string) {
	// This function is a placeholder for updating online status.
	// You can implement the logic to update the online status of users here.
	// For example, you might want to update a field in the database or send a message to a message queue.
	log.Println("Updating online status... (function not implemented yet)")
	redisKey := "online:" + id
	_ = config.Redis.Set(config.Ctx, redisKey, 1, 1*time.Minute).Err()

}
