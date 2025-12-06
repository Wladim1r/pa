package service

import (
	"github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	RegisterUser(name string, password string) error
	LoginUser(name string, password string) (*models.User, error)
	DeleteUserByID(userID uuid.UUID) error
}

type userService struct {
	userRepo repository.UsersDB
}

func NewUserService(userRepo repository.UsersDB) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) RegisterUser(name string, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &models.User{
		Name:     name,
		Password: string(hashedPassword),
	}
	return s.userRepo.CreateUser(user)
}

func (s *userService) LoginUser(name string, password string) (*models.User, error) {
	user, err := s.userRepo.GetUserByName(name)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errs.ErrRecordingWNF
	}
	return user, nil
}

func (s *userService) DeleteUserByID(userID uuid.UUID) error {
	return s.userRepo.DeleteUserByID(userID)
}
