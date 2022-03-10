package models

import (
	"gorm.io/gorm"
)

type PlacedBet struct {
	gorm.Model
	Trash            bool
	UserID           uint    `json:"user_id"`
	WalletID         uint    `json:"wallet_id"`
	RoundID          uint    `json:"round_id"`
	Status           uint8   `json:"status"`
	PayoutMultiplier float32 `json:"multiplier"`
	Amount           float32 `json:"amount"`
	// User             User   `gorm:"foreignKey:UserID"`
	// Wallet           Wallet `gorm:"foreignKey:WalletID"`
	// Round            Round  `gorm:"foreignKey:BetID"`
}
