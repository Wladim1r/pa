// Package models
package models

type User struct {
	ID       uint   `json:"id"       gorm:"primaryKey"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Request struct {
	Name     string `json:"name"     binding:"required"`
	Password string `json:"password" binding:"required"`
}
