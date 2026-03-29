package database

import (
	"x-ui/database/model"
	"x-ui/xray"
)

const (
	migrationProviderSQLite  = "sqlite"
	migrationProviderMongoDB = "mongodb"
)

type migrationSnapshot struct {
	Users            []*model.User
	Inbounds         []*model.Inbound
	ClientTraffics   []*xray.ClientTraffic
	Settings         []*model.Setting
	OutboundTraffics []*model.OutboundTraffics
	InboundClientIPs []*model.InboundClientIps
	HistoryOfSeeders []*model.HistoryOfSeeders
	LotteryWins      []*model.LotteryWin
	LinkHistories    []*LinkHistory
}

type migrationCounts struct {
	Users            int64
	Inbounds         int64
	ClientTraffics   int64
	Settings         int64
	OutboundTraffics int64
	InboundClientIPs int64
	HistoryOfSeeders int64
	LotteryWins      int64
	LinkHistories    int64
}

func (s migrationSnapshot) counts() migrationCounts {
	return migrationCounts{
		Users:            int64(len(s.Users)),
		Inbounds:         int64(len(s.Inbounds)),
		ClientTraffics:   int64(len(s.ClientTraffics)),
		Settings:         int64(len(s.Settings)),
		OutboundTraffics: int64(len(s.OutboundTraffics)),
		InboundClientIPs: int64(len(s.InboundClientIPs)),
		HistoryOfSeeders: int64(len(s.HistoryOfSeeders)),
		LotteryWins:      int64(len(s.LotteryWins)),
		LinkHistories:    int64(len(s.LinkHistories)),
	}
}
