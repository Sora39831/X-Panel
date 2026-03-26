package service

import (
	"encoding/json"
	"errors"
	"time"

	"x-ui/config"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/util/crypto"
	"x-ui/xray"

	"github.com/google/uuid"
	"github.com/xlzd/gotp"
	"gorm.io/gorm"
)

type UserService struct {
	settingService SettingService
}

func (s *UserService) GetFirstUser() (*model.User, error) {
	if config.GetDBType() == "mongodb" {
		return database.GetProvider().GetFirstUser()
	}
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		First(user).
		Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) CheckUser(username string, password string, twoFactorCode string) *model.User {
	var user *model.User
	var err error

	if config.GetDBType() == "mongodb" {
		user, err = database.GetProvider().GetUserByUsername(username)
		if database.GetProvider().IsNotFound(err) {
			return nil
		}
	} else {
		db := database.GetDB()
		user = &model.User{}
		err = db.Model(model.User{}).
			Where("username = ?", username).
			First(user).
			Error
		if err == gorm.ErrRecordNotFound {
			return nil
		}
	}
	if err != nil {
		logger.Warning("check user err:", err)
		return nil
	}

	if !crypto.CheckPasswordHash(user.Password, password) {
		return nil
	}

	twoFactorEnable, err := s.settingService.GetTwoFactorEnable()
	if err != nil {
		logger.Warning("check two factor err:", err)
		return nil
	}

	if twoFactorEnable {
		twoFactorToken, err := s.settingService.GetTwoFactorToken()

		if err != nil {
			logger.Warning("check two factor token err:", err)
			return nil
		}

		if gotp.NewDefaultTOTP(twoFactorToken).Now() != twoFactorCode {
			return nil
		}
	}

	return user
}

func (s *UserService) UpdateUser(id int, username string, password string) error {
	hashedPassword, err := crypto.HashPasswordAsBcrypt(password)
	if err != nil {
		return err
	}

	twoFactorEnable, err := s.settingService.GetTwoFactorEnable()
	if err != nil {
		return err
	}

	if twoFactorEnable {
		s.settingService.SetTwoFactorEnable(false)
		s.settingService.SetTwoFactorToken("")
	}

	if config.GetDBType() == "mongodb" {
		return database.GetProvider().UpdateUserByID(id, map[string]any{"username": username, "password": hashedPassword})
	}

	db := database.GetDB()
	return db.Model(model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{"username": username, "password": hashedPassword}).
		Error
}

func (s *UserService) UpdateFirstUser(username string, password string) error {
	if username == "" {
		return errors.New("username can not be empty")
	} else if password == "" {
		return errors.New("password can not be empty")
	}
	hashedPassword, er := crypto.HashPasswordAsBcrypt(password)

	if er != nil {
		return er
	}

	if config.GetDBType() == "mongodb" {
		provider := database.GetProvider()
		user, err := provider.GetFirstUser()
		if provider.IsNotFound(err) {
			user = &model.User{}
			user.Username = username
			user.Password = hashedPassword
			return provider.CreateUser(user)
		} else if err != nil {
			return err
		}
		user.Username = username
		user.Password = hashedPassword
		return provider.SaveUser(user)
	}

	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).First(user).Error
	if database.IsNotFound(err) {
		user.Username = username
		user.Password = hashedPassword
		return db.Model(model.User{}).Create(user).Error
	} else if err != nil {
		return err
	}
	user.Username = username
	user.Password = hashedPassword
	return db.Save(user).Error
}

func (s *UserService) CheckEmailExists(email string) (bool, error) {
	// 检查 User 表
	if config.GetDBType() == "mongodb" {
		provider := database.GetProvider()
		_, err := provider.GetUserByUsername(email)
		if err == nil {
			return true, nil
		}
		if !provider.IsNotFound(err) {
			return false, err
		}
	} else {
		db := database.GetDB()
		var count int64
		err := db.Model(model.User{}).Where("username = ?", email).Count(&count).Error
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}

	// 检查 ClientTraffic 表
	inboundService := InboundService{}
	traffic, _ := inboundService.GetClientTrafficByEmail(email)
	if traffic != nil {
		return true, nil
	}

	return false, nil
}

func (s *UserService) Register(email string, password string) error {
	// 1. 检查 email 是否已存在
	exists, err := s.CheckEmailExists(email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("email already exists")
	}

	// 2. 生成 UUID 作为客户端 ID
	clientID := uuid.New().String()

	// 3. 生成密码 hash
	hashedPassword, err := crypto.HashPasswordAsBcrypt(password)
	if err != nil {
		return err
	}

	// 4. 遍历所有 inbound，为每个 inbound 添加客户端
	inboundService := InboundService{}
	inbounds, err := inboundService.GetAllInbounds()
	if err != nil {
		return err
	}

	for _, inbound := range inbounds {
		// 解析 Settings JSON
		var settings map[string]any
		err := json.Unmarshal([]byte(inbound.Settings), &settings)
		if err != nil {
			logger.Warning("Failed to parse inbound settings:", err)
			continue
		}

		// 获取 clients 数组
		clientsInterface, ok := settings["clients"]
		if !ok {
			continue
		}

		clientsSlice, ok := clientsInterface.([]any)
		if !ok {
			continue
		}

		// 创建新的 Client 对象
		nowTs := time.Now().Unix() * 1000
		newClient := map[string]any{
			"id":         clientID,
			"email":      email,
			"totalGB":    0,
			"expiryTime": 0,
			"enable":     true,
			"limitIp":    0,
			"flow":       "",
			"created_at": nowTs,
			"updated_at": nowTs,
			"reset":      0,
		}

		// 根据协议设置不同的字段
		switch inbound.Protocol {
		case "trojan":
			newClient["password"] = clientID
		case "shadowsocks":
			newClient["password"] = clientID
		default:
			// vless/vmess 使用 id 字段
		}

		// 追加到 clients 数组
		clientsSlice = append(clientsSlice, newClient)
		settings["clients"] = clientsSlice

		// 序列化回 JSON
		newSettings, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			logger.Warning("Failed to marshal inbound settings:", err)
			continue
		}
		inbound.Settings = string(newSettings)

		// 保存 inbound
		if config.GetDBType() == "mongodb" {
			if err := database.GetProvider().SaveInbound(inbound); err != nil {
				logger.Warning("Failed to save inbound:", err)
				continue
			}
		} else {
			if err := database.GetDB().Save(inbound).Error; err != nil {
				logger.Warning("Failed to save inbound:", err)
				continue
			}
		}
	}

	// 5. 创建唯一的 ClientTraffic 记录（Email 字段有 gorm:"unique" 约束，每个 email 只能有一条记录）
	// 使用第一个 inbound 的 ID 作为关联，如果没有 inbound 则使用 0
	firstInboundId := 0
	if len(inbounds) > 0 {
		firstInboundId = inbounds[0].Id
	}
	clientTraffic := xray.ClientTraffic{}
	clientTraffic.InboundId = firstInboundId
	clientTraffic.Email = email
	clientTraffic.Total = 0
	clientTraffic.ExpiryTime = 0
	clientTraffic.Enable = true
	clientTraffic.Up = 0
	clientTraffic.Down = 0

	if config.GetDBType() == "mongodb" {
		if err := database.GetProvider().CreateClientTraffic(&clientTraffic); err != nil {
			logger.Warning("Failed to create client traffic:", err)
		}
	} else {
		if err := database.GetDB().Create(&clientTraffic).Error; err != nil {
			logger.Warning("Failed to create client traffic:", err)
		}
	}

	// 6. 创建 User 记录
	user := &model.User{
		Username: email,
		Password: hashedPassword,
		Role:     "user",
	}

	if config.GetDBType() == "mongodb" {
		return database.GetProvider().CreateUser(user)
	}
	return database.GetDB().Create(user).Error
}
