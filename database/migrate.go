package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"x-ui/database/model"
	"x-ui/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MigrateSQLiteToMongoDB copies all data from SQLite to MongoDB.
func MigrateSQLiteToMongoDB(sqliteDBPath string) error {
	mongoProvider, ok := provider.(*MongoDBProvider)
	if !ok {
		return fmt.Errorf("current provider is not MongoDB")
	}

	if _, err := os.Stat(sqliteDBPath); os.IsNotExist(err) {
		return fmt.Errorf("SQLite database not found at %s", sqliteDBPath)
	}

	sqliteDB, err := gorm.Open(sqlite.Open(sqliteDBPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.Info("Starting SQLite to MongoDB migration...")

	if err := migrateUsers(sqliteDB, mongoProvider, ctx); err != nil {
		return fmt.Errorf("migrate users: %w", err)
	}

	if err := migrateSettings(sqliteDB, mongoProvider, ctx); err != nil {
		return fmt.Errorf("migrate settings: %w", err)
	}

	if err := migrateInbounds(sqliteDB, mongoProvider, ctx); err != nil {
		return fmt.Errorf("migrate inbounds: %w", err)
	}

	if err := migrateOutboundTraffics(sqliteDB, mongoProvider, ctx); err != nil {
		return fmt.Errorf("migrate outbound traffics: %w", err)
	}

	if err := migrateLinkHistory(sqliteDB, mongoProvider, ctx); err != nil {
		return fmt.Errorf("migrate link history: %w", err)
	}

	logger.Info("SQLite to MongoDB migration completed successfully")
	return nil
}

func migrateUsers(sqliteDB *gorm.DB, p *MongoDBProvider, ctx context.Context) error {
	var users []*model.User
	if err := sqliteDB.Model(model.User{}).Find(&users).Error; err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	var maxID int
	for _, u := range users {
		if u.Id > maxID {
			maxID = u.Id
		}
		if _, err := p.db.Collection("users").InsertOne(ctx, u); err != nil {
			return err
		}
	}
	return p.syncCounter(ctx, "users", maxID)
}

func migrateSettings(sqliteDB *gorm.DB, p *MongoDBProvider, ctx context.Context) error {
	var settings []*model.Setting
	if err := sqliteDB.Model(model.Setting{}).Find(&settings).Error; err != nil {
		return err
	}
	if len(settings) == 0 {
		return nil
	}
	docs := make([]interface{}, len(settings))
	for i, s := range settings {
		docs[i] = s
	}
	_, err := p.db.Collection("settings").InsertMany(ctx, docs)
	return err
}

func migrateInbounds(sqliteDB *gorm.DB, p *MongoDBProvider, ctx context.Context) error {
	var inbounds []*model.Inbound
	if err := sqliteDB.Model(model.Inbound{}).Preload("ClientStats").Find(&inbounds).Error; err != nil {
		return err
	}
	if len(inbounds) == 0 {
		return nil
	}
	var maxID int
	for _, in := range inbounds {
		if in.Id > maxID {
			maxID = in.Id
		}
		if _, err := p.db.Collection("inbounds").InsertOne(ctx, in); err != nil {
			return err
		}
		for _, ct := range in.ClientStats {
			if _, err := p.db.Collection("client_traffics").InsertOne(ctx, ct); err != nil {
				return err
			}
		}
	}
	return p.syncCounter(ctx, "inbounds", maxID)
}

func migrateOutboundTraffics(sqliteDB *gorm.DB, p *MongoDBProvider, ctx context.Context) error {
	var traffics []*model.OutboundTraffics
	if err := sqliteDB.Model(model.OutboundTraffics{}).Find(&traffics).Error; err != nil {
		return err
	}
	if len(traffics) == 0 {
		return nil
	}
	var maxID int
	for _, t := range traffics {
		if t.Id > maxID {
			maxID = t.Id
		}
		if _, err := p.db.Collection("outbound_traffics").InsertOne(ctx, t); err != nil {
			return err
		}
	}
	return p.syncCounter(ctx, "outbound_traffics", maxID)
}

func migrateLinkHistory(sqliteDB *gorm.DB, p *MongoDBProvider, ctx context.Context) error {
	var history []*LinkHistory
	err := sqliteDB.Order("created_at desc").Limit(10).Find(&history).Error
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return nil
	}
	docs := make([]interface{}, len(history))
	for i, h := range history {
		docs[i] = h
	}
	_, err = p.db.Collection("link_histories").InsertMany(ctx, docs)
	return err
}

func (p *MongoDBProvider) syncCounter(ctx context.Context, collection string, maxID int) error {
	_, err := p.counters.UpdateOne(
		ctx,
		bson.M{"_id": collection},
		bson.M{"$set": bson.M{"seq": maxID}},
		options.Update().SetUpsert(true),
	)
	return err
}
