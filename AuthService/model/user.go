package model

type User struct {
	Id          uint   `gorm:"primaryKey"`
	Phone       string `gorm:"unique"`
	Password    string `gorm:"not null"`
	Name        string
	Avatar      string
	Background  string
	Description string `gorm:"default:''"`
	Gender      uint   `gorm:"default:1"`
	Birth       string
	Permission  uint `gorm:"default:1"`
}
