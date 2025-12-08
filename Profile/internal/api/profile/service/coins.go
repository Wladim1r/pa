package service

import (
	"github.com/Wladim1r/profile/internal/models"
	"github.com/shopspring/decimal"
)

type CoinsService interface {
	GetCoins(userID float64) ([]*models.Coin, error)
	AddCoin(userID float64, symbol string, quantity float32) error
	UpdateCoin(userID float64, symbol string, quantity float32) error
	DeleteCoin(userID float64, symbol string) error
}

func (ps *service) GetCoins(userID int) ([]*models.Coin, error) {
	return ps.cr.GetCoins(uint(userID))
}

func (ps *service) AddCoin(userID int, symbol string, quantity float32) error {
	q := decimal.NewFromFloat32(quantity)

	coin := models.Coin{Symbol: symbol, Quantity: q, UserID: uint(userID)}

	return ps.cr.AddCoin(&coin)
}

func (ps *service) UpdateCoin(userID int, symbol string, quantity float32) error {
	q := decimal.NewFromFloat32(quantity)

	return ps.cr.UpdateCoin(uint(userID), symbol, q)
}

func (ps *service) DeleteCoin(userID int, symbol string) error {
	return ps.cr.DeleteCoin(uint(userID), symbol)
}
