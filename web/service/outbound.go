package service

import (
	"x-ui/config"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/xray"

	"gorm.io/gorm"
)

type OutboundService struct{}

func (s *OutboundService) AddTraffic(traffics []*xray.Traffic, clientTraffics []*xray.ClientTraffic) (error, bool) {
	var err error

	if config.GetDBType() == "mongodb" {
		provider := database.GetProvider()
		tx, err := provider.BeginTransaction()
		if err != nil {
			return err, false
		}
		defer func() {
			if err != nil {
				tx.RollbackTransaction()
			} else {
				tx.CommitTransaction()
			}
		}()

		for _, traffic := range traffics {
			if traffic.IsOutbound {
				outbound, ferr := tx.FirstOrCreateOutboundTraffic(traffic.Tag)
				if ferr != nil {
					err = ferr
					return err, false
				}
				outbound.Tag = traffic.Tag
				outbound.Up = outbound.Up + traffic.Up
				outbound.Down = outbound.Down + traffic.Down
				outbound.Total = outbound.Up + outbound.Down
				if ferr = tx.SaveOutboundTraffic(outbound); ferr != nil {
					err = ferr
					return err, false
				}
			}
		}

		return nil, false
	}

	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = s.addOutboundTraffic(tx, traffics)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (s *OutboundService) addOutboundTraffic(tx *gorm.DB, traffics []*xray.Traffic) error {
	if len(traffics) == 0 {
		return nil
	}

	var err error

	for _, traffic := range traffics {
		if traffic.IsOutbound {

			var outbound model.OutboundTraffics

			err = tx.Model(&model.OutboundTraffics{}).Where("tag = ?", traffic.Tag).
				FirstOrCreate(&outbound).Error
			if err != nil {
				return err
			}

			outbound.Tag = traffic.Tag
			outbound.Up = outbound.Up + traffic.Up
			outbound.Down = outbound.Down + traffic.Down
			outbound.Total = outbound.Up + outbound.Down

			err = tx.Save(&outbound).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *OutboundService) GetOutboundsTraffic() ([]*model.OutboundTraffics, error) {
	if config.GetDBType() == "mongodb" {
		traffics, err := database.GetProvider().GetOutboundTraffics()
		if err != nil {
			logger.Warning("Error retrieving OutboundTraffics: ", err)
			return nil, err
		}
		return traffics, nil
	}

	db := database.GetDB()
	var traffics []*model.OutboundTraffics

	err := db.Model(model.OutboundTraffics{}).Find(&traffics).Error
	if err != nil {
		logger.Warning("Error retrieving OutboundTraffics: ", err)
		return nil, err
	}

	return traffics, nil
}

func (s *OutboundService) ResetOutboundTraffic(tag string) error {
	if config.GetDBType() == "mongodb" {
		allTags := tag == "-alltags-"
		return database.GetProvider().ResetOutboundTraffics(tag, allTags)
	}

	db := database.GetDB()

	whereText := "tag "
	if tag == "-alltags-" {
		whereText += " <> ?"
	} else {
		whereText += " = ?"
	}

	result := db.Model(model.OutboundTraffics{}).
		Where(whereText, tag).
		Updates(map[string]any{"up": 0, "down": 0, "total": 0})

	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
