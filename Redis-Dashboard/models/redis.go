// Package models
package models

type SecondStat struct {
	EventTime int64   `json:"E"` // Время когда сервер отправил
	Symbol    string  `json:"s"` // Торговая пара
	Price     float64 `json:"p"` // Цена сделки
	Quantity  string  `json:"q"` // Объем сделки
	TradeTime int64   `json:"T"` // Время самой сделки
}
