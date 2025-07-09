package config

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB
var Saltid int = 1

func ConnectDB() {
	srcdbconnect := "root:123456@tcp(localhost:3307)/mydb?charset=utf8mb4&parseTime=True&loc=Asia%2FHo_Chi_Minh"
	db, err := gorm.Open(mysql.Open(srcdbconnect), &gorm.Config{})
	if err != nil {
		panic("Kết nối DB lỗi: " + err.Error())
	}
	log.Println("Kết nối DB OK")

	DB = db
}
func Taobanguser() error {
	result := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			phone VARCHAR(100) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255),
			avatar TEXT ,
			background TEXT ,
			description TEXT ,
			gender TINYINT DEFAULT 1,
			birth VARCHAR(20), -- Dùng VARCHAR để lưu ngày tháng dưới dạng chuỗi,
			permission TINYINT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if result.Error != nil {
		panic("Lỗi tạo bảng users: " + result.Error.Error())
	}
	log.Println("Tạo bảng user OK")
	return nil
}
func Taobangtoken() error {
	result := DB.Exec(`
		CREATE TABLE IF NOT EXISTS tokens (
			id INT,
			refreshtoken VARCHAR(255),
			accesstoken TEXT,
			deviceinfo TEXT, 
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY(id , refreshtoken),
			FOREIGN KEY (id) REFERENCES users (id) ON DELETE CASCADE
		)
	`)
	if result.Error != nil {
		panic("Lỗi tạo bảng token: " + result.Error.Error())
	}
	log.Println("Tạo bảng token OK")
	return nil
}
func Taobangpost() error {
	result := DB.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			postid INT AUTO_INCREMENT PRIMARY KEY,
			id INT,
			content TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (id) REFERENCES users (id) ON DELETE CASCADE
		)
	`)
	if result.Error != nil {
		panic("Lỗi tạo bảng post: " + result.Error.Error())
	}
	log.Println("Tạo bảng post OK")
	return nil
}
func Taobangchitietpost() error {
	result := DB.Exec(`
		CREATE TABLE IF NOT EXISTS chitietposts (
			mediaid INT AUTO_INCREMENT ,
			postid INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY(mediaid , postid),
			FOREIGN KEY (postid) REFERENCES posts (postid) ON DELETE CASCADE
		)
	`)
	if result.Error != nil {
		panic("Lỗi tạo bảng chitietpost: " + result.Error.Error())
	}
	log.Println("Tạo bảng chitietpost OK")
	return nil
}
