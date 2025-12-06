package repository

import "gorm.io/gorm"

type repository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) (UsersRepository, CoinsRepository) {
	repo := &repository{db: db}
	return repo, repo
}
