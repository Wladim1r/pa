package service

import (
	"fmt"
	"strconv"

	"github.com/Wladim1r/profile/internal/models"
)

type UsersService interface {
	CreateUserProfile(userID string, name string) error
	GetUserProfileByUserID(userID float64) (*models.User, error)
	DeleteUserProfileByUserID(userID float64) error
	CheckUserProfileExists(userID int) (bool, error)
}

func (r *service) CreateUserProfile(userIDstr string, name string) error {
	userID, err := strconv.Atoi(userIDstr)
	if err != nil {
		return fmt.Errorf("failed to convert 'userID' from string into int: %w", err)
	}

	user := models.User{
		ID:   uint(userID),
		Name: name,
	}

	return r.ur.CreateUserProfile(user)
}

func (r *service) GetUserProfileByUserID(userID int) (*models.User, error) {
	return r.ur.GetUserProfileByUserID(uint(userID))
}

func (r *service) DeleteUserProfileByUserID(userID float64) error {
	return r.ur.DeleteUserProfileByUserID(uint(userID))
}

func (r *service) CheckUserProfileExists(userID int) (bool, error) {
	return r.ur.CheckUserProfileExists(uint(userID))
}
