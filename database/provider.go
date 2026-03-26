package database

import (
	"io"
	"x-ui/database/model"
	"x-ui/xray"

	"gorm.io/gorm"
)

type DBProvider interface {
	// === Lifecycle ===
	Init(dbPath string) error
	Close() error

	// === Common query ===
	IsNotFound(err error) bool

	// === User operations ===
	GetFirstUser() (*model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	CreateUser(user *model.User) error
	UpdateUserByID(id int, updates map[string]any) error
	SaveUser(user *model.User) error
	DeleteUserByUsername(username string) error
	GetAllUsers() ([]*model.User, error)

	// === Inbound operations ===
	GetInboundsWithClientStats() ([]*model.Inbound, error)
	GetAllInboundsWithClientStats() ([]*model.Inbound, error)
	GetInboundByID(id int) (*model.Inbound, error)
	CreateInbound(inbound *model.Inbound) error
	SaveInbound(inbound *model.Inbound) error
	DeleteInboundByID(id int) error
	GetInboundTagByID(id int) (string, error)
	GetInboundIDs() ([]int, error)
	CountInboundsByPort(port int) (int64, error)
	GetInboundEmails() ([]string, error)

	// === Transaction ===
	BeginTransaction() (DBProvider, error)
	CommitTransaction() error
	RollbackTransaction() error

	// === Client traffic ===
	CreateClientTraffic(traffic *xray.ClientTraffic) error
	SaveClientTraffic(traffic *xray.ClientTraffic) error
	GetClientTrafficsByEmails(emails []string) ([]*xray.ClientTraffic, error)
	GetClientTrafficsByIDs(ids []int) ([]*xray.ClientTraffic, error)
	DeleteClientTrafficByID(id int) error
	UpdateClientTrafficByEmail(email string, up, down int64) error
	UpdateClientTrafficsBatch(traffics []*xray.ClientTraffic) error
	SelectClientTrafficEnableByEmail(email string) (bool, error)

	// === Client IPs ===
	GetAllInboundClientIps() ([]*model.InboundClientIps, error)
	GetInboundClientIpsByEmail(clientEmail string) (*model.InboundClientIps, error)
	SaveInboundClientIps(ips *model.InboundClientIps) error
	DeleteClientIpsByEmail(clientEmail string) error
	ClearClientIpsByEmail(clientEmail string) error

	// === Outbound traffic ===
	GetOutboundTraffics() ([]*model.OutboundTraffics, error)
	FirstOrCreateOutboundTraffic(tag string) (*model.OutboundTraffics, error)
	SaveOutboundTraffic(traffic *model.OutboundTraffics) error
	ResetOutboundTraffics(tag string, allTags bool) error

	// === Settings ===
	GetAllSettings() ([]*model.Setting, error)
	GetSettingByKey(key string) (*model.Setting, error)
	CreateSetting(setting *model.Setting) error
	SaveSetting(setting *model.Setting) error
	DeleteAllSettings() error

	// === Link history ===
	AddLinkHistory(record *LinkHistory) error
	GetLinkHistory() ([]*LinkHistory, error)

	// === Lottery ===
	HasUserWonToday(userID int64) (bool, error)
	RecordUserWin(userID int64, prize string) error

	// === Seeder ===
	GetSeederNames() ([]string, error)
	CreateSeederHistory(name string) error
	IsTableEmpty(tableName string) (bool, error)

	// === Advanced inbound ===
	DisableInvalidInbounds(expiryTime int64) (int64, error)
	DisableInvalidClients(expiryTime int64) (int64, error)
	MigrationRemoveOrphanedTraffics() error

	// === SQLite-only (MongoDB: no-op or returns nil) ===
	Checkpoint() error
	IsSQLiteDB(file io.ReaderAt) (bool, error)
	GetGormDB() *gorm.DB
}
