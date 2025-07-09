package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	AppPort  string
	DBUser   string
	DBPass   string
	DBName   string
	RedisURL string
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Không tìm thấy file .env, dùng biến môi trường hệ thống")
	}

	AppPort = getEnv("APP_PORT", "8080")
	DBUser = getEnv("DB_USER", "root")
	DBPass = getEnv("DB_PASS", "")
	DBName = getEnv("DB_NAME", "testdb")
	RedisURL = getEnv("REDIS_URL", "localhost:6379")

}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
