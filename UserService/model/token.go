package model

type Token struct {
	Id           uint
	Refreshtoken string
	Accesstoken  string
	Deviceinfo   string
	Created_at   string `gorm:"default:CURRENT_TIMESTAMP"`
}
