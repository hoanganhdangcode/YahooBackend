package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
)

func main() {
	// Tạo private key (2048 bit)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Encode và lưu private key ra file
	privFile, err := os.Create("private.pem")
	if err != nil {
		panic(err)
	}
	defer privFile.Close()

	privateBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pem.Encode(privFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateBytes,
	})

	// Encode và lưu public key ra file
	pubFile, err := os.Create("public.pem")
	if err != nil {
		panic(err)
	}
	defer pubFile.Close()

	publicBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}
	pem.Encode(pubFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	})

	println("✅ Đã tạo xong private.pem và public.pem")

	key := make([]byte, 32) // 32 bytes = 256 bit
	_, err = rand.Read(key)
	if err != nil {
		panic(err)
	}

	// Encode base64 để dễ lưu, dùng luôn trong code nếu muốn
	encoded := base64.StdEncoding.EncodeToString(key)

	// Ghi vào file
	err = os.WriteFile("aes.key", []byte(encoded), 0644)
	if err != nil {
		panic(err)
	}

	println("✅ AES-256 key (base64) đã được tạo và lưu vào aes.key")
}
