package handler

import (
	"UserService/config"
	"UserService/model"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

// var Saltid int = 1

// func Gensalt() (string, error) {
// 	bytes := make([]byte, 8) // 16 bytes = 128 bits
// 	_, err := rand.Read(bytes)
// 	if err != nil {
// 		Saltid++
// 		return "salt" + strconv.Itoa(Saltid), err
// 	}
// 	return hex.EncodeToString(bytes), nil
// }

//	func HashMD5(password, salt string) string {
//		data := []byte(password + salt)
//		hash := md5.Sum(data)
//		return hex.EncodeToString(hash[:])
//	}
func loadPrivateKey() (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile("../genkey/private.pem")
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(keyData)
}
func GenerateJWT(userID uint) (string, error) {
	privateKey, err := loadPrivateKey()
	if err != nil {
		return "", err
	}
	var user model.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("user not found")
	}
	tsecond := time.Now().Unix()
	jti := uuid.New().String()
	claims := jwt.MapClaims{
		"sub":  userID,
		"jti":  jti,
		"role": user.Permission,
		"exp":  tsecond + 60*15,
		"iat":  tsecond,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 256-bit
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
