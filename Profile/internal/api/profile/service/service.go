package service

import "github.com/Wladim1r/profile/internal/api/profile/repository"

type service struct {
	ur repository.UsersRepository
	cr repository.CoinsRepository
}

func NewProfileService(
	ur repository.UsersRepository,
	cr repository.CoinsRepository,
) (UsersService, CoinsService) {
	s := &service{ur: ur, cr: cr}

	return s, s
}
