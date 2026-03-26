package model

import "time"

// LotteryWin 用于记录用户的中奖历史
type LotteryWin struct {
	ID        int64     `gorm:"primaryKey" bson:"_id,omitempty"`
	UserID    int64     `gorm:"index" bson:"user_id"` // Telegram 用户 ID
	Prize     string    `bson:"prize"`                 // 奖品等级，如 "一等奖"
	WinDate   time.Time `bson:"win_date"`              // 中奖日期
}