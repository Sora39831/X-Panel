package database

import (
	"io"
	"x-ui/database/model"
	"x-ui/xray"

	"gorm.io/gorm"
)

type SQLiteProvider struct{}

func (p *SQLiteProvider) Init(dbPath string) error {
	return InitDB(dbPath)
}

func (p *SQLiteProvider) Close() error {
	return CloseDB()
}

func (p *SQLiteProvider) IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

// === User ===

func (p *SQLiteProvider) GetFirstUser() (*model.User, error) {
	user := &model.User{}
	err := db.Model(model.User{}).First(user).Error
	return user, err
}

func (p *SQLiteProvider) GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := db.Model(model.User{}).Where("username = ?", username).First(user).Error
	return user, err
}

func (p *SQLiteProvider) CreateUser(user *model.User) error {
	return db.Model(model.User{}).Create(user).Error
}

func (p *SQLiteProvider) UpdateUserByID(id int, updates map[string]any) error {
	return db.Model(model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (p *SQLiteProvider) SaveUser(user *model.User) error {
	return db.Save(user).Error
}

func (p *SQLiteProvider) DeleteUserByUsername(username string) error {
	return db.Where("username = ?", username).Delete(model.User{}).Error
}

func (p *SQLiteProvider) GetAllUsers() ([]*model.User, error) {
	var users []*model.User
	err := db.Model(model.User{}).Find(&users).Error
	return users, err
}

// === Inbound ===

func (p *SQLiteProvider) GetInboundsWithClientStats() ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Preload("ClientStats").Find(&inbounds).Error
	return inbounds, err
}

func (p *SQLiteProvider) GetAllInboundsWithClientStats() ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	err := db.Model(model.Inbound{}).Preload("ClientStats").Find(&inbounds).Error
	return inbounds, err
}

func (p *SQLiteProvider) GetInboundByID(id int) (*model.Inbound, error) {
	inbound := &model.Inbound{}
	err := db.Model(model.Inbound{}).First(inbound, id).Error
	return inbound, err
}

func (p *SQLiteProvider) CreateInbound(inbound *model.Inbound) error {
	return db.Create(inbound).Error
}

func (p *SQLiteProvider) SaveInbound(inbound *model.Inbound) error {
	return db.Save(inbound).Error
}

func (p *SQLiteProvider) DeleteInboundByID(id int) error {
	return db.Delete(model.Inbound{}, id).Error
}

func (p *SQLiteProvider) GetInboundTagByID(id int) (string, error) {
	var tag string
	err := db.Model(model.Inbound{}).Select("tag").Where("id = ?", id).First(&tag).Error
	return tag, err
}

func (p *SQLiteProvider) GetInboundIDs() ([]int, error) {
	var ids []int
	err := db.Model(model.Inbound{}).Pluck("id", &ids).Error
	return ids, err
}

func (p *SQLiteProvider) CountInboundsByPort(port int) (int64, error) {
	var count int64
	err := db.Model(model.Inbound{}).Where("port = ?", port).Count(&count).Error
	return count, err
}

func (p *SQLiteProvider) GetInboundEmails() ([]string, error) {
	var emails []string
	err := db.Raw("SELECT DISTINCT value FROM inbounds, json_each(JSON_EXTRACT(settings, '$.clients')) WHERE json_extract(value, '$.email') IS NOT NULL").Scan(&emails).Error
	return emails, err
}

// === Transaction ===

type SQLiteTransaction struct {
	tx *gorm.DB
}

func (p *SQLiteProvider) BeginTransaction() (DBProvider, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &SQLiteTransaction{tx: tx}, nil
}

func (t *SQLiteTransaction) Init(dbPath string) error                  { return nil }
func (t *SQLiteTransaction) Close() error                              { return nil }
func (t *SQLiteTransaction) IsNotFound(err error) bool                 { return err == gorm.ErrRecordNotFound }
func (t *SQLiteTransaction) CommitTransaction() error                  { return t.tx.Commit().Error }
func (t *SQLiteTransaction) RollbackTransaction() error                { return t.tx.Rollback().Error }
func (t *SQLiteTransaction) Checkpoint() error                         { return nil }
func (t *SQLiteTransaction) IsSQLiteDB(file io.ReaderAt) (bool, error) { return IsSQLiteDB(file) }
func (t *SQLiteTransaction) GetGormDB() *gorm.DB                      { return t.tx }

func (t *SQLiteTransaction) BeginTransaction() (DBProvider, error) {
	return nil, nil
}

// Transaction User methods

func (t *SQLiteTransaction) GetFirstUser() (*model.User, error) {
	user := &model.User{}
	err := t.tx.Model(model.User{}).First(user).Error
	return user, err
}

func (t *SQLiteTransaction) GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := t.tx.Model(model.User{}).Where("username = ?", username).First(user).Error
	return user, err
}

func (t *SQLiteTransaction) CreateUser(user *model.User) error {
	return t.tx.Model(model.User{}).Create(user).Error
}

func (t *SQLiteTransaction) UpdateUserByID(id int, updates map[string]any) error {
	return t.tx.Model(model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (t *SQLiteTransaction) SaveUser(user *model.User) error {
	return t.tx.Save(user).Error
}

func (t *SQLiteTransaction) DeleteUserByUsername(username string) error {
	return t.tx.Where("username = ?", username).Delete(model.User{}).Error
}

func (t *SQLiteTransaction) GetAllUsers() ([]*model.User, error) {
	var users []*model.User
	err := t.tx.Model(model.User{}).Find(&users).Error
	return users, err
}

// Transaction Inbound methods

func (t *SQLiteTransaction) GetInboundsWithClientStats() ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	err := t.tx.Model(model.Inbound{}).Preload("ClientStats").Find(&inbounds).Error
	return inbounds, err
}

func (t *SQLiteTransaction) GetAllInboundsWithClientStats() ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	err := t.tx.Model(model.Inbound{}).Preload("ClientStats").Find(&inbounds).Error
	return inbounds, err
}

func (t *SQLiteTransaction) GetInboundByID(id int) (*model.Inbound, error) {
	inbound := &model.Inbound{}
	err := t.tx.Model(model.Inbound{}).First(inbound, id).Error
	return inbound, err
}

func (t *SQLiteTransaction) CreateInbound(inbound *model.Inbound) error {
	return t.tx.Create(inbound).Error
}

func (t *SQLiteTransaction) SaveInbound(inbound *model.Inbound) error {
	return t.tx.Save(inbound).Error
}

func (t *SQLiteTransaction) DeleteInboundByID(id int) error {
	return t.tx.Delete(model.Inbound{}, id).Error
}

func (t *SQLiteTransaction) GetInboundTagByID(id int) (string, error) {
	var tag string
	err := t.tx.Model(model.Inbound{}).Select("tag").Where("id = ?", id).First(&tag).Error
	return tag, err
}

func (t *SQLiteTransaction) GetInboundIDs() ([]int, error) {
	var ids []int
	err := t.tx.Model(model.Inbound{}).Pluck("id", &ids).Error
	return ids, err
}

func (t *SQLiteTransaction) CountInboundsByPort(port int) (int64, error) {
	var count int64
	err := t.tx.Model(model.Inbound{}).Where("port = ?", port).Count(&count).Error
	return count, err
}

func (t *SQLiteTransaction) GetInboundEmails() ([]string, error) {
	var emails []string
	err := t.tx.Raw("SELECT DISTINCT value FROM inbounds, json_each(JSON_EXTRACT(settings, '$.clients')) WHERE json_extract(value, '$.email') IS NOT NULL").Scan(&emails).Error
	return emails, err
}

// === Client Traffic ===

func (p *SQLiteProvider) CreateClientTraffic(traffic *xray.ClientTraffic) error {
	return db.Create(traffic).Error
}

func (p *SQLiteProvider) SaveClientTraffic(traffic *xray.ClientTraffic) error {
	return db.Save(traffic).Error
}

func (p *SQLiteProvider) GetClientTrafficsByEmails(emails []string) ([]*xray.ClientTraffic, error) {
	var traffics []*xray.ClientTraffic
	err := db.Model(xray.ClientTraffic{}).Where("email IN ?", emails).Find(&traffics).Error
	return traffics, err
}

func (p *SQLiteProvider) GetClientTrafficsByIDs(ids []int) ([]*xray.ClientTraffic, error) {
	var traffics []*xray.ClientTraffic
	err := db.Model(xray.ClientTraffic{}).Where("id IN ?", ids).Find(&traffics).Error
	return traffics, err
}

func (p *SQLiteProvider) DeleteClientTrafficByID(id int) error {
	return db.Delete(xray.ClientTraffic{}, id).Error
}

func (p *SQLiteProvider) UpdateClientTrafficByEmail(email string, up, down int64) error {
	return db.Model(xray.ClientTraffic{}).Where("email = ?", email).Updates(map[string]any{
		"up":   gorm.Expr("up + ?", up),
		"down": gorm.Expr("down + ?", down),
	}).Error
}

func (p *SQLiteProvider) UpdateClientTrafficsBatch(traffics []*xray.ClientTraffic) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, traffic := range traffics {
			if err := tx.Model(xray.ClientTraffic{}).Where("email = ?", traffic.Email).Updates(map[string]any{
				"up":   gorm.Expr("up + ?", traffic.Up),
				"down": gorm.Expr("down + ?", traffic.Down),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *SQLiteProvider) SelectClientTrafficEnableByEmail(email string) (bool, error) {
	var enable bool
	err := db.Model(xray.ClientTraffic{}).Select("enable").Where("email = ?", email).First(&enable).Error
	return enable, err
}

// Transaction Client Traffic methods

func (t *SQLiteTransaction) CreateClientTraffic(traffic *xray.ClientTraffic) error {
	return t.tx.Create(traffic).Error
}

func (t *SQLiteTransaction) SaveClientTraffic(traffic *xray.ClientTraffic) error {
	return t.tx.Save(traffic).Error
}

func (t *SQLiteTransaction) GetClientTrafficsByEmails(emails []string) ([]*xray.ClientTraffic, error) {
	var traffics []*xray.ClientTraffic
	err := t.tx.Model(xray.ClientTraffic{}).Where("email IN ?", emails).Find(&traffics).Error
	return traffics, err
}

func (t *SQLiteTransaction) GetClientTrafficsByIDs(ids []int) ([]*xray.ClientTraffic, error) {
	var traffics []*xray.ClientTraffic
	err := t.tx.Model(xray.ClientTraffic{}).Where("id IN ?", ids).Find(&traffics).Error
	return traffics, err
}

func (t *SQLiteTransaction) DeleteClientTrafficByID(id int) error {
	return t.tx.Delete(xray.ClientTraffic{}, id).Error
}

func (t *SQLiteTransaction) UpdateClientTrafficByEmail(email string, up, down int64) error {
	return t.tx.Model(xray.ClientTraffic{}).Where("email = ?", email).Updates(map[string]any{
		"up":   gorm.Expr("up + ?", up),
		"down": gorm.Expr("down + ?", down),
	}).Error
}

func (t *SQLiteTransaction) UpdateClientTrafficsBatch(traffics []*xray.ClientTraffic) error {
	for _, traffic := range traffics {
		if err := t.tx.Model(xray.ClientTraffic{}).Where("email = ?", traffic.Email).Updates(map[string]any{
			"up":   gorm.Expr("up + ?", traffic.Up),
			"down": gorm.Expr("down + ?", traffic.Down),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (t *SQLiteTransaction) SelectClientTrafficEnableByEmail(email string) (bool, error) {
	var enable bool
	err := t.tx.Model(xray.ClientTraffic{}).Select("enable").Where("email = ?", email).First(&enable).Error
	return enable, err
}

// === Client IPs ===

func (p *SQLiteProvider) GetAllInboundClientIps() ([]*model.InboundClientIps, error) {
	var ips []*model.InboundClientIps
	err := db.Model(model.InboundClientIps{}).Find(&ips).Error
	return ips, err
}

func (p *SQLiteProvider) GetInboundClientIpsByEmail(clientEmail string) (*model.InboundClientIps, error) {
	ip := &model.InboundClientIps{}
	err := db.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).First(ip).Error
	return ip, err
}

func (p *SQLiteProvider) SaveInboundClientIps(ips *model.InboundClientIps) error {
	return db.Save(ips).Error
}

func (p *SQLiteProvider) DeleteClientIpsByEmail(clientEmail string) error {
	return db.Where("client_email = ?", clientEmail).Delete(model.InboundClientIps{}).Error
}

func (p *SQLiteProvider) ClearClientIpsByEmail(clientEmail string) error {
	return db.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).Update("ips", "").Error
}

// Transaction Client IPs methods

func (t *SQLiteTransaction) GetAllInboundClientIps() ([]*model.InboundClientIps, error) {
	var ips []*model.InboundClientIps
	err := t.tx.Model(model.InboundClientIps{}).Find(&ips).Error
	return ips, err
}

func (t *SQLiteTransaction) GetInboundClientIpsByEmail(clientEmail string) (*model.InboundClientIps, error) {
	ip := &model.InboundClientIps{}
	err := t.tx.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).First(ip).Error
	return ip, err
}

func (t *SQLiteTransaction) SaveInboundClientIps(ips *model.InboundClientIps) error {
	return t.tx.Save(ips).Error
}

func (t *SQLiteTransaction) DeleteClientIpsByEmail(clientEmail string) error {
	return t.tx.Where("client_email = ?", clientEmail).Delete(model.InboundClientIps{}).Error
}

func (t *SQLiteTransaction) ClearClientIpsByEmail(clientEmail string) error {
	return t.tx.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).Update("ips", "").Error
}

// === Outbound Traffic ===

func (p *SQLiteProvider) GetOutboundTraffics() ([]*model.OutboundTraffics, error) {
	var traffics []*model.OutboundTraffics
	err := db.Model(model.OutboundTraffics{}).Find(&traffics).Error
	return traffics, err
}

func (p *SQLiteProvider) FirstOrCreateOutboundTraffic(tag string) (*model.OutboundTraffics, error) {
	traffic := &model.OutboundTraffics{}
	err := db.Model(model.OutboundTraffics{}).Where("tag = ?", tag).First(traffic).Error
	if err == gorm.ErrRecordNotFound {
		traffic.Tag = tag
		return traffic, db.Create(traffic).Error
	}
	return traffic, err
}

func (p *SQLiteProvider) SaveOutboundTraffic(traffic *model.OutboundTraffics) error {
	return db.Save(traffic).Error
}

func (p *SQLiteProvider) ResetOutboundTraffics(tag string, allTags bool) error {
	if allTags {
		return db.Model(model.OutboundTraffics{}).Where("1 = 1").Updates(map[string]any{
			"up":    0,
			"down":  0,
			"total": 0,
		}).Error
	}
	return db.Model(model.OutboundTraffics{}).Where("tag = ?", tag).Updates(map[string]any{
		"up":    0,
		"down":  0,
		"total": 0,
	}).Error
}

// Transaction Outbound Traffic methods

func (t *SQLiteTransaction) GetOutboundTraffics() ([]*model.OutboundTraffics, error) {
	var traffics []*model.OutboundTraffics
	err := t.tx.Model(model.OutboundTraffics{}).Find(&traffics).Error
	return traffics, err
}

func (t *SQLiteTransaction) FirstOrCreateOutboundTraffic(tag string) (*model.OutboundTraffics, error) {
	traffic := &model.OutboundTraffics{}
	err := t.tx.Model(model.OutboundTraffics{}).Where("tag = ?", tag).First(traffic).Error
	if err == gorm.ErrRecordNotFound {
		traffic.Tag = tag
		return traffic, t.tx.Create(traffic).Error
	}
	return traffic, err
}

func (t *SQLiteTransaction) SaveOutboundTraffic(traffic *model.OutboundTraffics) error {
	return t.tx.Save(traffic).Error
}

func (t *SQLiteTransaction) ResetOutboundTraffics(tag string, allTags bool) error {
	if allTags {
		return t.tx.Model(model.OutboundTraffics{}).Where("1 = 1").Updates(map[string]any{
			"up":    0,
			"down":  0,
			"total": 0,
		}).Error
	}
	return t.tx.Model(model.OutboundTraffics{}).Where("tag = ?", tag).Updates(map[string]any{
		"up":    0,
		"down":  0,
		"total": 0,
	}).Error
}

// === Settings ===

func (p *SQLiteProvider) GetAllSettings() ([]*model.Setting, error) {
	var settings []*model.Setting
	err := db.Model(model.Setting{}).Find(&settings).Error
	return settings, err
}

func (p *SQLiteProvider) GetSettingByKey(key string) (*model.Setting, error) {
	setting := &model.Setting{}
	err := db.Model(model.Setting{}).Where("key = ?", key).First(setting).Error
	return setting, err
}

func (p *SQLiteProvider) CreateSetting(setting *model.Setting) error {
	return db.Create(setting).Error
}

func (p *SQLiteProvider) SaveSetting(setting *model.Setting) error {
	return db.Save(setting).Error
}

func (p *SQLiteProvider) DeleteAllSettings() error {
	return db.Where("1 = 1").Delete(model.Setting{}).Error
}

// Transaction Settings methods

func (t *SQLiteTransaction) GetAllSettings() ([]*model.Setting, error) {
	var settings []*model.Setting
	err := t.tx.Model(model.Setting{}).Find(&settings).Error
	return settings, err
}

func (t *SQLiteTransaction) GetSettingByKey(key string) (*model.Setting, error) {
	setting := &model.Setting{}
	err := t.tx.Model(model.Setting{}).Where("key = ?", key).First(setting).Error
	return setting, err
}

func (t *SQLiteTransaction) CreateSetting(setting *model.Setting) error {
	return t.tx.Create(setting).Error
}

func (t *SQLiteTransaction) SaveSetting(setting *model.Setting) error {
	return t.tx.Save(setting).Error
}

func (t *SQLiteTransaction) DeleteAllSettings() error {
	return t.tx.Where("1 = 1").Delete(model.Setting{}).Error
}

// === Link History ===

func (p *SQLiteProvider) AddLinkHistory(record *LinkHistory) error {
	return AddLinkHistory(record)
}

func (p *SQLiteProvider) GetLinkHistory() ([]*LinkHistory, error) {
	return GetLinkHistory()
}

// Transaction Link History methods

func (t *SQLiteTransaction) AddLinkHistory(record *LinkHistory) error {
	return t.tx.Create(record).Error
}

func (t *SQLiteTransaction) GetLinkHistory() ([]*LinkHistory, error) {
	var history []*LinkHistory
	err := t.tx.Order("created_at desc").Limit(10).Find(&history).Error
	return history, err
}

// === Lottery ===

func (p *SQLiteProvider) HasUserWonToday(userID int64) (bool, error) {
	return HasUserWonToday(userID)
}

func (p *SQLiteProvider) RecordUserWin(userID int64, prize string) error {
	return RecordUserWin(userID, prize)
}

// Transaction Lottery methods

func (t *SQLiteTransaction) HasUserWonToday(userID int64) (bool, error) {
	return HasUserWonToday(userID)
}

func (t *SQLiteTransaction) RecordUserWin(userID int64, prize string) error {
	return RecordUserWin(userID, prize)
}

// === Seeder ===

func (p *SQLiteProvider) GetSeederNames() ([]string, error) {
	var names []string
	err := db.Model(model.HistoryOfSeeders{}).Pluck("seeder_name", &names).Error
	return names, err
}

func (p *SQLiteProvider) CreateSeederHistory(name string) error {
	return db.Create(&model.HistoryOfSeeders{SeederName: name}).Error
}

func (p *SQLiteProvider) IsTableEmpty(tableName string) (bool, error) {
	return isTableEmpty(tableName)
}

// Transaction Seeder methods

func (t *SQLiteTransaction) GetSeederNames() ([]string, error) {
	var names []string
	err := t.tx.Model(model.HistoryOfSeeders{}).Pluck("seeder_name", &names).Error
	return names, err
}

func (t *SQLiteTransaction) CreateSeederHistory(name string) error {
	return t.tx.Create(&model.HistoryOfSeeders{SeederName: name}).Error
}

func (t *SQLiteTransaction) IsTableEmpty(tableName string) (bool, error) {
	var count int64
	err := t.tx.Table(tableName).Count(&count).Error
	return count == 0, err
}

// === Advanced inbound ===

func (p *SQLiteProvider) DisableInvalidInbounds(expiryTime int64) (int64, error) {
	result := db.Model(model.Inbound{}).
		Where("enable = ? AND expiry_time > ? AND expiry_time < ?", true, 0, expiryTime).
		Update("enable", false)
	return result.RowsAffected, result.Error
}

func (p *SQLiteProvider) DisableInvalidClients(expiryTime int64) (int64, error) {
	result := db.Model(xray.ClientTraffic{}).
		Where("enable = ? AND expiry_time > ? AND expiry_time < ?", true, 0, expiryTime).
		Update("enable", false)
	return result.RowsAffected, result.Error
}

func (p *SQLiteProvider) MigrationRemoveOrphanedTraffics() error {
	return db.Exec("DELETE FROM client_traffics WHERE id NOT IN (SELECT DISTINCT value FROM inbounds, json_each(JSON_EXTRACT(settings, '$.clients')), json_each((SELECT '[' || group_concat(json_extract(value, '$.email')) || ']' FROM json_each(JSON_EXTRACT(settings, '$.clients')))) WHERE client_traffics.email = json_each.value)").Error
}

// Transaction Advanced inbound methods

func (t *SQLiteTransaction) DisableInvalidInbounds(expiryTime int64) (int64, error) {
	result := t.tx.Model(model.Inbound{}).
		Where("enable = ? AND expiry_time > ? AND expiry_time < ?", true, 0, expiryTime).
		Update("enable", false)
	return result.RowsAffected, result.Error
}

func (t *SQLiteTransaction) DisableInvalidClients(expiryTime int64) (int64, error) {
	result := t.tx.Model(xray.ClientTraffic{}).
		Where("enable = ? AND expiry_time > ? AND expiry_time < ?", true, 0, expiryTime).
		Update("enable", false)
	return result.RowsAffected, result.Error
}

func (t *SQLiteTransaction) MigrationRemoveOrphanedTraffics() error {
	return t.tx.Exec("DELETE FROM client_traffics WHERE id NOT IN (SELECT DISTINCT value FROM inbounds, json_each(JSON_EXTRACT(settings, '$.clients')), json_each((SELECT '[' || group_concat(json_extract(value, '$.email')) || ']' FROM json_each(JSON_EXTRACT(settings, '$.clients')))) WHERE client_traffics.email = json_each.value)").Error
}

// Provider GetGormDB
func (p *SQLiteProvider) GetGormDB() *gorm.DB {
	return GetDB()
}

func (p *SQLiteProvider) CommitTransaction() error                  { return nil }
func (p *SQLiteProvider) RollbackTransaction() error                { return nil }
func (p *SQLiteProvider) Checkpoint() error                         { return Checkpoint() }
func (p *SQLiteProvider) IsSQLiteDB(file io.ReaderAt) (bool, error) { return IsSQLiteDB(file) }
