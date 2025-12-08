// Package models
package models

import (
	"github.com/shopspring/decimal"
)

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"unique;not null"`
	Coins []Coin `gorm:"foreignKey:UserID"`
}

type Coin struct {
	ID       uint            `gorm:"primaryKey"`
	Symbol   string          `gorm:"not null;unique"`
	Quantity decimal.Decimal `gorm:"type:decimal(20,8);not null"`

	UserID uint
	User   User `gorm:"constraint:OnDelete:CASCADE;"`
}

type UserRequest struct {
	Name     string `json:"name"     binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CoinRequest struct {
	Symbol   string  `json:"symbol"   binding:"required"`
	Quantity float32 `json:"quantity" binding:"required"`
}

// type SecondStat struct {
// 	EventTime int64   `json:"E"` // Время когда сервер отправил
// 	Symbol    string  `json:"s"` // Торговая пара
// 	Price     float64 `json:"p"` // Цена сделки
// 	Quantity  string  `json:"q"` // Объем сделки
// 	TradeTime int64   `json:"T"` // Время самой сделки
// }
//
// type MarketTicker struct {
// 	MessageID     string          `db:"message_id"`
// 	EventType     string          `db:"event_type"`
// 	EventTime     time.Time       `db:"event_time"`
// 	ReceiveTime   time.Time       `db:"receive_time"`
// 	Symbol        string          `db:"symbol"`
// 	ClosePrice    decimal.Decimal `db:"close_price"`
// 	OpenPrice     decimal.Decimal `db:"open_price"`
// 	HighPrice     decimal.Decimal `db:"high_price"`
// 	LowPrice      decimal.Decimal `db:"low_price"`
// 	ChangePrice   decimal.Decimal `db:"change_price"`
// 	ChangePercent decimal.Decimal `db:"change_percent"`
// }
