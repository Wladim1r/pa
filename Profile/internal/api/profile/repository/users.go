package repository

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/Wladim1r/profile/internal/models"
	"github.com/Wladim1r/profile/lib/errs"
	"gorm.io/gorm"
)

type UsersRepository interface {
	CreateTables()
	CreateUserProfile(user models.User) error
	GetUserProfileByUserID(userID uint) (*models.User, error)
	DeleteUserProfileByUserID(userID uint) error
	CheckUserProfileExists(userID uint) (bool, error)
}

func (pr *repository) CreateTables() {
	// create tables
	if err := pr.db.AutoMigrate(&models.User{}, &models.Coin{}); err != nil {
		slog.Error("Could not create db table", "error", err.Error())
		os.Exit(1)
	}

	// check table exists or not
	// if ok := pr.db.Migrator().HasTable("user_profilies"); !ok {
	// 	slog.Error("Table 'user_profilies' has not created idk")
	// 	os.Exit(1)
	// }
	if ok := pr.db.Migrator().HasTable("coins"); !ok {
		slog.Error("Table 'coins' has not created idk")
		os.Exit(1)
	}
}

func (r *repository) CreateUserProfile(user models.User) error {
	if err := r.db.Create(&user).Error; err != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return nil
}

func (r *repository) GetUserProfileByUserID(userID uint) (*models.User, error) {
	var user models.User
	if err := r.db.Preload("Coins").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", errs.ErrRecordingWNF, err.Error())
		}
		return nil, fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return &user, nil
}

func (r *repository) DeleteUserProfileByUserID(userID uint) error {
	if err := r.db.Where("id = ?", userID).Delete(&models.User{}).Error; err != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return nil
}

func (r *repository) CheckUserProfileExists(userID uint) (bool, error) {
	return true, nil
}
