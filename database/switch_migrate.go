package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"x-ui/database/model"
	"x-ui/xray"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var migrationMongoCounterCollections = []string{
	"users",
	"inbounds",
	"client_traffics",
	"outbound_traffics",
	"settings",
	"history_of_seeders",
	"link_histories",
	"lottery_wins",
}

func MigrateBetweenProviders(fromType, toType, sqlitePath string) error {
	fromType = normalizeMigrationProviderType(fromType)
	toType = normalizeMigrationProviderType(toType)

	if fromType == "" || toType == "" {
		return fmt.Errorf("both --from and --to are required and must be one of: sqlite, mongodb")
	}
	if fromType == toType {
		return fmt.Errorf("source and target providers must be different")
	}

	snapshot, err := loadMigrationSnapshot(fromType, sqlitePath)
	if err != nil {
		return err
	}

	switch toType {
	case migrationProviderSQLite:
		return writeSnapshotToSQLite(snapshot, sqlitePath)
	case migrationProviderMongoDB:
		return writeSnapshotToMongoDB(snapshot)
	default:
		return fmt.Errorf("unsupported target provider: %s", toType)
	}
}

func normalizeMigrationProviderType(providerType string) string {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case migrationProviderSQLite:
		return migrationProviderSQLite
	case migrationProviderMongoDB:
		return migrationProviderMongoDB
	default:
		return ""
	}
}

func loadMigrationSnapshot(providerType, sqlitePath string) (migrationSnapshot, error) {
	switch providerType {
	case migrationProviderSQLite:
		return loadSnapshotFromSQLite(sqlitePath)
	case migrationProviderMongoDB:
		return loadSnapshotFromMongoDB()
	default:
		return migrationSnapshot{}, fmt.Errorf("unsupported source provider: %s", providerType)
	}
}

func loadSnapshotFromSQLite(sqlitePath string) (migrationSnapshot, error) {
	if _, err := os.Stat(sqlitePath); err != nil {
		return migrationSnapshot{}, fmt.Errorf("failed to access sqlite database %s: %w", sqlitePath, err)
	}

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	if err != nil {
		return migrationSnapshot{}, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	snapshot := migrationSnapshot{}
	if err := db.Order("id asc").Find(&snapshot.Users).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load users from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.Inbounds).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load inbounds from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.ClientTraffics).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load client_traffics from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.Settings).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load settings from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.OutboundTraffics).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load outbound_traffics from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.InboundClientIPs).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load inbound_client_ips from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.HistoryOfSeeders).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load history_of_seeders from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.LotteryWins).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load lottery_wins from sqlite: %w", err)
	}
	if err := db.Order("id asc").Find(&snapshot.LinkHistories).Error; err != nil {
		return migrationSnapshot{}, fmt.Errorf("load link_histories from sqlite: %w", err)
	}

	return snapshot, nil
}

func loadSnapshotFromMongoDB() (migrationSnapshot, error) {
	provider := &MongoDBProvider{}
	if err := provider.Init(""); err != nil {
		return migrationSnapshot{}, fmt.Errorf("failed to connect mongodb source: %w", err)
	}
	defer provider.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	snapshot := migrationSnapshot{}
	if err := loadMongoCollection(ctx, provider.db.Collection("users"), &snapshot.Users); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load users from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("inbounds"), &snapshot.Inbounds); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load inbounds from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("client_traffics"), &snapshot.ClientTraffics); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load client_traffics from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("settings"), &snapshot.Settings); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load settings from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("outbound_traffics"), &snapshot.OutboundTraffics); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load outbound_traffics from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("inbound_client_ips"), &snapshot.InboundClientIPs); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load inbound_client_ips from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("history_of_seeders"), &snapshot.HistoryOfSeeders); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load history_of_seeders from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("lottery_wins"), &snapshot.LotteryWins); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load lottery_wins from mongodb: %w", err)
	}
	if err := loadMongoCollection(ctx, provider.db.Collection("link_histories"), &snapshot.LinkHistories); err != nil {
		return migrationSnapshot{}, fmt.Errorf("load link_histories from mongodb: %w", err)
	}

	return snapshot, nil
}

func writeSnapshotToSQLite(snapshot migrationSnapshot, sqlitePath string) error {
	if err := os.MkdirAll(filepath.Dir(sqlitePath), 0o755); err != nil {
		return fmt.Errorf("failed to create sqlite directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open sqlite target: %w", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	if err := autoMigrateSwitchTables(db); err != nil {
		return fmt.Errorf("failed to prepare sqlite schema: %w", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := clearSQLiteMigrationTables(tx); err != nil {
			return err
		}
		return insertSnapshotIntoSQLite(tx, snapshot)
	}); err != nil {
		return err
	}

	targetCounts, err := countSQLiteSnapshot(db)
	if err != nil {
		return fmt.Errorf("count sqlite target: %w", err)
	}
	if err := verifyMigrationCounts(snapshot.counts(), targetCounts); err != nil {
		return err
	}

	return nil
}

func writeSnapshotToMongoDB(snapshot migrationSnapshot) error {
	provider := &MongoDBProvider{}
	if err := provider.Init(""); err != nil {
		return fmt.Errorf("failed to connect mongodb target: %w", err)
	}
	defer provider.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := clearMongoMigrationCollections(ctx, provider.db); err != nil {
		return err
	}
	if err := insertSnapshotIntoMongo(ctx, provider.db, snapshot); err != nil {
		return err
	}

	targetCounts, err := countMongoSnapshot(ctx, provider.db)
	if err != nil {
		return fmt.Errorf("count mongodb target: %w", err)
	}
	if err := verifyMigrationCounts(snapshot.counts(), targetCounts); err != nil {
		return err
	}

	if err := syncMongoCountersForSnapshot(ctx, provider, snapshot); err != nil {
		return err
	}

	return nil
}

func autoMigrateSwitchTables(db *gorm.DB) error {
	models := []any{
		&model.User{},
		&model.Inbound{},
		&xray.ClientTraffic{},
		&model.Setting{},
		&model.OutboundTraffics{},
		&model.InboundClientIps{},
		&model.HistoryOfSeeders{},
		&model.LotteryWin{},
		&LinkHistory{},
	}
	return db.AutoMigrate(models...)
}

func clearSQLiteMigrationTables(tx *gorm.DB) error {
	for _, table := range []string{
		"client_traffics",
		"inbound_client_ips",
		"lottery_wins",
		"history_of_seeders",
		"link_histories",
		"outbound_traffics",
		"inbounds",
		"settings",
		"users",
	} {
		if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
			return fmt.Errorf("clear sqlite table %s: %w", table, err)
		}
	}
	return nil
}

func insertSnapshotIntoSQLite(tx *gorm.DB, snapshot migrationSnapshot) error {
	if err := createInBatches(tx, snapshot.Users); err != nil {
		return fmt.Errorf("insert users into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.Inbounds); err != nil {
		return fmt.Errorf("insert inbounds into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.ClientTraffics); err != nil {
		return fmt.Errorf("insert client_traffics into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.Settings); err != nil {
		return fmt.Errorf("insert settings into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.OutboundTraffics); err != nil {
		return fmt.Errorf("insert outbound_traffics into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.InboundClientIPs); err != nil {
		return fmt.Errorf("insert inbound_client_ips into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.HistoryOfSeeders); err != nil {
		return fmt.Errorf("insert history_of_seeders into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.LotteryWins); err != nil {
		return fmt.Errorf("insert lottery_wins into sqlite: %w", err)
	}
	if err := createInBatches(tx, snapshot.LinkHistories); err != nil {
		return fmt.Errorf("insert link_histories into sqlite: %w", err)
	}
	return nil
}

func countSQLiteSnapshot(db *gorm.DB) (migrationCounts, error) {
	counts := migrationCounts{}
	if err := db.Model(&model.User{}).Count(&counts.Users).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&model.Inbound{}).Count(&counts.Inbounds).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&xray.ClientTraffic{}).Count(&counts.ClientTraffics).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&model.Setting{}).Count(&counts.Settings).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&model.OutboundTraffics{}).Count(&counts.OutboundTraffics).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&model.InboundClientIps{}).Count(&counts.InboundClientIPs).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&model.HistoryOfSeeders{}).Count(&counts.HistoryOfSeeders).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&model.LotteryWin{}).Count(&counts.LotteryWins).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&LinkHistory{}).Count(&counts.LinkHistories).Error; err != nil {
		return counts, err
	}
	return counts, nil
}

func clearMongoMigrationCollections(ctx context.Context, db *mongo.Database) error {
	for _, collection := range []string{
		"users",
		"inbounds",
		"client_traffics",
		"settings",
		"outbound_traffics",
		"inbound_client_ips",
		"history_of_seeders",
		"lottery_wins",
		"link_histories",
	} {
		if _, err := db.Collection(collection).DeleteMany(ctx, bson.M{}); err != nil {
			return fmt.Errorf("clear mongodb collection %s: %w", collection, err)
		}
	}

	if _, err := db.Collection("counters").DeleteMany(ctx, bson.M{"_id": bson.M{"$in": migrationMongoCounterCollections}}); err != nil {
		return fmt.Errorf("clear mongodb counters: %w", err)
	}

	return nil
}

func insertSnapshotIntoMongo(ctx context.Context, db *mongo.Database, snapshot migrationSnapshot) error {
	if err := insertMongoDocuments(ctx, db.Collection("users"), snapshot.Users); err != nil {
		return fmt.Errorf("insert users into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("inbounds"), snapshot.Inbounds); err != nil {
		return fmt.Errorf("insert inbounds into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("client_traffics"), snapshot.ClientTraffics); err != nil {
		return fmt.Errorf("insert client_traffics into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("settings"), snapshot.Settings); err != nil {
		return fmt.Errorf("insert settings into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("outbound_traffics"), snapshot.OutboundTraffics); err != nil {
		return fmt.Errorf("insert outbound_traffics into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("inbound_client_ips"), snapshot.InboundClientIPs); err != nil {
		return fmt.Errorf("insert inbound_client_ips into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("history_of_seeders"), snapshot.HistoryOfSeeders); err != nil {
		return fmt.Errorf("insert history_of_seeders into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("lottery_wins"), snapshot.LotteryWins); err != nil {
		return fmt.Errorf("insert lottery_wins into mongodb: %w", err)
	}
	if err := insertMongoDocuments(ctx, db.Collection("link_histories"), snapshot.LinkHistories); err != nil {
		return fmt.Errorf("insert link_histories into mongodb: %w", err)
	}
	return nil
}

func countMongoSnapshot(ctx context.Context, db *mongo.Database) (migrationCounts, error) {
	counts := migrationCounts{}

	var err error
	if counts.Users, err = db.Collection("users").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.Inbounds, err = db.Collection("inbounds").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.ClientTraffics, err = db.Collection("client_traffics").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.Settings, err = db.Collection("settings").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.OutboundTraffics, err = db.Collection("outbound_traffics").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.InboundClientIPs, err = db.Collection("inbound_client_ips").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.HistoryOfSeeders, err = db.Collection("history_of_seeders").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.LotteryWins, err = db.Collection("lottery_wins").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}
	if counts.LinkHistories, err = db.Collection("link_histories").CountDocuments(ctx, bson.M{}); err != nil {
		return counts, err
	}

	return counts, nil
}

func verifyMigrationCounts(expected, actual migrationCounts) error {
	if expected.Users != actual.Users {
		return fmt.Errorf("users count mismatch: source=%d target=%d", expected.Users, actual.Users)
	}
	if expected.Inbounds != actual.Inbounds {
		return fmt.Errorf("inbounds count mismatch: source=%d target=%d", expected.Inbounds, actual.Inbounds)
	}
	if expected.ClientTraffics != actual.ClientTraffics {
		return fmt.Errorf("client_traffics count mismatch: source=%d target=%d", expected.ClientTraffics, actual.ClientTraffics)
	}
	if expected.Settings != actual.Settings {
		return fmt.Errorf("settings count mismatch: source=%d target=%d", expected.Settings, actual.Settings)
	}
	if expected.OutboundTraffics != actual.OutboundTraffics {
		return fmt.Errorf("outbound_traffics count mismatch: source=%d target=%d", expected.OutboundTraffics, actual.OutboundTraffics)
	}
	if expected.InboundClientIPs != actual.InboundClientIPs {
		return fmt.Errorf("inbound_client_ips count mismatch: source=%d target=%d", expected.InboundClientIPs, actual.InboundClientIPs)
	}
	if expected.HistoryOfSeeders != actual.HistoryOfSeeders {
		return fmt.Errorf("history_of_seeders count mismatch: source=%d target=%d", expected.HistoryOfSeeders, actual.HistoryOfSeeders)
	}
	if expected.LotteryWins != actual.LotteryWins {
		return fmt.Errorf("lottery_wins count mismatch: source=%d target=%d", expected.LotteryWins, actual.LotteryWins)
	}
	if expected.LinkHistories != actual.LinkHistories {
		return fmt.Errorf("link_histories count mismatch: source=%d target=%d", expected.LinkHistories, actual.LinkHistories)
	}
	return nil
}

func syncMongoCountersForSnapshot(ctx context.Context, provider *MongoDBProvider, snapshot migrationSnapshot) error {
	if err := provider.syncCounter(ctx, "users", maxIntValue(snapshot.Users, func(user *model.User) int {
		return user.Id
	})); err != nil {
		return fmt.Errorf("sync users counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "inbounds", maxIntValue(snapshot.Inbounds, func(inbound *model.Inbound) int {
		return inbound.Id
	})); err != nil {
		return fmt.Errorf("sync inbounds counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "client_traffics", maxIntValue(snapshot.ClientTraffics, func(traffic *xray.ClientTraffic) int {
		return traffic.Id
	})); err != nil {
		return fmt.Errorf("sync client_traffics counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "outbound_traffics", maxIntValue(snapshot.OutboundTraffics, func(traffic *model.OutboundTraffics) int {
		return traffic.Id
	})); err != nil {
		return fmt.Errorf("sync outbound_traffics counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "settings", maxIntValue(snapshot.Settings, func(setting *model.Setting) int {
		return setting.Id
	})); err != nil {
		return fmt.Errorf("sync settings counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "history_of_seeders", maxIntValue(snapshot.HistoryOfSeeders, func(history *model.HistoryOfSeeders) int {
		return history.Id
	})); err != nil {
		return fmt.Errorf("sync history_of_seeders counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "link_histories", maxIntValue(snapshot.LinkHistories, func(history *LinkHistory) int {
		return history.Id
	})); err != nil {
		return fmt.Errorf("sync link_histories counter: %w", err)
	}
	if err := provider.syncCounter(ctx, "lottery_wins", maxInt64Value(snapshot.LotteryWins, func(history *model.LotteryWin) int64 {
		return history.ID
	})); err != nil {
		return fmt.Errorf("sync lottery_wins counter: %w", err)
	}
	return nil
}

func loadMongoCollection[T any](ctx context.Context, collection *mongo.Collection, result *[]*T) error {
	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var records []*T
	if err := cursor.All(ctx, &records); err != nil {
		return err
	}

	*result = records
	return nil
}

func insertMongoDocuments[T any](ctx context.Context, collection *mongo.Collection, records []*T) error {
	if len(records) == 0 {
		return nil
	}

	documents := make([]any, 0, len(records))
	for _, record := range records {
		documents = append(documents, record)
	}

	_, err := collection.InsertMany(ctx, documents)
	return err
}

func createInBatches[T any](db *gorm.DB, records []*T) error {
	if len(records) == 0 {
		return nil
	}
	return db.CreateInBatches(records, 200).Error
}

func maxIntValue[T any](records []*T, getValue func(*T) int) int {
	maxValue := 0
	for _, record := range records {
		if value := getValue(record); value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func maxInt64Value[T any](records []*T, getValue func(*T) int64) int {
	maxValue := int64(0)
	for _, record := range records {
		if value := getValue(record); value > maxValue {
			maxValue = value
		}
	}
	return int(maxValue)
}
