package main

import (
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
)

// Hàm tạo reverse proxy đến backend
func reverseProxy(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetURL, _ := url.Parse(target)
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		c.Request.URL.Path = c.Param("proxyPath")
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	r := gin.Default()

	// Route đến service A (ví dụ localhost:8081)
	r.Any("/service-a/*proxyPath", reverseProxy("http://localhost:8081"))

	// Route đến service B (ví dụ localhost:8082)
	r.Any("/service-b/*proxyPath", reverseProxy("http://localhost:8082"))

	r.Run(":8080") // Gateway chạy ở cổng 8080
}

var secretKey = "hello"

func generateToken(userID string, role string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"role":   role,
		"exp":    time.Now().Add(time.Hour * 24).Unix(), // Hết hạn sau 24h
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(secretKey)
}
