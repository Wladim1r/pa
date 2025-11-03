package models

import (
	"log/slog"
	"strconv"
)

type MiniTicker struct {
	EventType     string `json:"e"` // "24hrMiniTicker"
	EventTime     int64  `json:"E"` // Время отправки
	Symbol        string `json:"s"` // Торговая пара
	ClosePrice    string `json:"c"` // Текущая цена
	OpenPrice     string `json:"o"` // Цена 24ч назад
	HighPrice     string `json:"h"` // Максимум за 24ч
	LowPrice      string `json:"l"` // Минимум за 24ч
	TotalBaseVol  string `json:"v"` // Объем (base asset)
	TotalQuoteVol string `json:"q"` // Объем (quote asset)
}

func (mk *MiniTicker) ClosePriceFloat() float64 {
	clPrice, err := strconv.ParseFloat(mk.ClosePrice, 64)
	if err != nil {
		slog.Error("Could not parse ClosePrice into float64", "error", err)
		return 0
	}
	return clPrice
}

func (mk *MiniTicker) OpenPriceFloat() float64 {
	opPrice, err := strconv.ParseFloat(mk.OpenPrice, 64)
	if err != nil {
		slog.Error("Could not parse OpenPrice into float64", "error", err)
		return 0
	}
	return opPrice
}
func (mk *MiniTicker) HighPriceFloat() float64 {
	hiPrice, err := strconv.ParseFloat(mk.HighPrice, 64)
	if err != nil {
		slog.Error("Could not parse HighPrice into float64", "error", err)
		return 0
	}
	return hiPrice
}
func (mk *MiniTicker) LowPriceFloat() float64 {
	loPrice, err := strconv.ParseFloat(mk.LowPrice, 64)
	if err != nil {
		slog.Error("Could not parse LowPrice into float64", "error", err)
		return 0
	}
	return loPrice
}
