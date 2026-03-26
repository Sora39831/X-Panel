package database

import (
	"context"
	"fmt"
	"io"
	"time"

	"x-ui/config"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/xray"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

type MongoDBProvider struct {
	client   *mongo.Client
	db       *mongo.Database
	counters *mongo.Collection
}

func (p *MongoDBProvider) Init(dbPath string) error {
	uri := config.GetMongoURI()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri).
		SetMaxPoolSize(100).
		SetMinPoolSize(5).
		SetMaxConnIdleTime(30 * time.Minute)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("mongodb connect failed: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb ping failed: %w", err)
	}

	p.client = client
	p.db = client.Database(config.GetMongoDBName())
	p.counters = p.db.Collection("counters")

	logger.Info("MongoDB connected successfully")
	return nil
}

func (p *MongoDBProvider) Close() error {
	if p.client != nil {
		return p.client.Disconnect(context.Background())
	}
	return nil
}

func (p *MongoDBProvider) IsNotFound(err error) bool {
	return err == mongo.ErrNoDocuments
}

func (p *MongoDBProvider) getNextSequence(collection string) (int, error) {
	filter := bson.M{"_id": collection}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	result := p.counters.FindOneAndUpdate(context.TODO(), filter, update, opts)
	var doc struct {
		Seq int `bson:"seq"`
	}
	if err := result.Decode(&doc); err != nil {
		return 0, err
	}
	return doc.Seq, nil
}

// === User ===

func (p *MongoDBProvider) GetFirstUser() (*model.User, error) {
	user := &model.User{}
	err := p.db.Collection("users").FindOne(context.TODO(), bson.M{}).Decode(user)
	return user, err
}

func (p *MongoDBProvider) GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := p.db.Collection("users").FindOne(context.TODO(), bson.M{"username": username}).Decode(user)
	return user, err
}

func (p *MongoDBProvider) CreateUser(user *model.User) error {
	id, err := p.getNextSequence("users")
	if err != nil {
		return err
	}
	user.Id = id
	_, err = p.db.Collection("users").InsertOne(context.TODO(), user)
	return err
}

func (p *MongoDBProvider) UpdateUserByID(id int, updates map[string]any) error {
	_, err := p.db.Collection("users").UpdateOne(context.TODO(), bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (p *MongoDBProvider) SaveUser(user *model.User) error {
	_, err := p.db.Collection("users").ReplaceOne(context.TODO(), bson.M{"_id": user.Id}, user)
	return err
}

func (p *MongoDBProvider) DeleteUserByUsername(username string) error {
	_, err := p.db.Collection("users").DeleteOne(context.TODO(), bson.M{"username": username})
	return err
}

func (p *MongoDBProvider) GetAllUsers() ([]*model.User, error) {
	cursor, err := p.db.Collection("users").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var users []*model.User
	if err := cursor.All(context.TODO(), &users); err != nil {
		return nil, err
	}
	return users, nil
}

// === Inbound ===

func (p *MongoDBProvider) GetInboundsWithClientStats() ([]*model.Inbound, error) {
	cursor, err := p.db.Collection("inbounds").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var inbounds []*model.Inbound
	if err := cursor.All(context.TODO(), &inbounds); err != nil {
		return nil, err
	}
	return p.populateClientStats(inbounds)
}

func (p *MongoDBProvider) GetAllInboundsWithClientStats() ([]*model.Inbound, error) {
	return p.GetInboundsWithClientStats()
}

func (p *MongoDBProvider) populateClientStats(inbounds []*model.Inbound) ([]*model.Inbound, error) {
	if len(inbounds) == 0 {
		return inbounds, nil
	}

	// Collect all inbound IDs
	inboundIDs := make([]int, len(inbounds))
	for i, inbound := range inbounds {
		inboundIDs[i] = inbound.Id
	}

	// Batch load client_traffics by inbound_id
	cursor, err := p.db.Collection("client_traffics").Find(context.TODO(), bson.M{"inbound_id": bson.M{"$in": inboundIDs}})
	if err != nil {
		return inbounds, err
	}
	defer cursor.Close(context.TODO())

	var traffics []*xray.ClientTraffic
	if err := cursor.All(context.TODO(), &traffics); err != nil {
		return inbounds, err
	}

	// Map traffics by inbound_id
	trafficMap := make(map[int][]xray.ClientTraffic)
	for _, t := range traffics {
		trafficMap[t.InboundId] = append(trafficMap[t.InboundId], *t)
	}

	// Assign to inbounds
	for _, inbound := range inbounds {
		if stats, ok := trafficMap[inbound.Id]; ok {
			inbound.ClientStats = stats
		} else {
			inbound.ClientStats = []xray.ClientTraffic{}
		}
	}

	return inbounds, nil
}

func (p *MongoDBProvider) GetInboundByID(id int) (*model.Inbound, error) {
	inbound := &model.Inbound{}
	err := p.db.Collection("inbounds").FindOne(context.TODO(), bson.M{"_id": id}).Decode(inbound)
	return inbound, err
}

func (p *MongoDBProvider) CreateInbound(inbound *model.Inbound) error {
	id, err := p.getNextSequence("inbounds")
	if err != nil {
		return err
	}
	inbound.Id = id
	_, err = p.db.Collection("inbounds").InsertOne(context.TODO(), inbound)
	return err
}

func (p *MongoDBProvider) SaveInbound(inbound *model.Inbound) error {
	_, err := p.db.Collection("inbounds").ReplaceOne(context.TODO(), bson.M{"_id": inbound.Id}, inbound)
	return err
}

func (p *MongoDBProvider) DeleteInboundByID(id int) error {
	_, err := p.db.Collection("inbounds").DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

func (p *MongoDBProvider) GetInboundTagByID(id int) (string, error) {
	var result struct {
		Tag string `bson:"tag"`
	}
	err := p.db.Collection("inbounds").FindOne(context.TODO(), bson.M{"_id": id}).Decode(&result)
	return result.Tag, err
}

func (p *MongoDBProvider) GetInboundIDs() ([]int, error) {
	results, err := p.db.Collection("inbounds").Distinct(context.TODO(), "_id", bson.M{})
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(results))
	for i, v := range results {
		switch val := v.(type) {
		case int32:
			ids[i] = int(val)
		case int64:
			ids[i] = int(val)
		case int:
			ids[i] = val
		default:
			ids[i] = 0
		}
	}
	return ids, nil
}

func (p *MongoDBProvider) CountInboundsByPort(port int) (int64, error) {
	return p.db.Collection("inbounds").CountDocuments(context.TODO(), bson.M{"port": port})
}

func (p *MongoDBProvider) GetInboundEmails() ([]string, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"settings.clients": bson.M{"$exists": true}}}},
		{{Key: "$unwind", Value: "$settings.clients"}},
		{{Key: "$match", Value: bson.M{"settings.clients.email": bson.M{"$ne": ""}}}},
		{{Key: "$group", Value: bson.M{"_id": "$settings.clients.email"}}},
	}
	cursor, err := p.db.Collection("inbounds").Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	emails := make([]string, len(results))
	for i, r := range results {
		emails[i] = r.ID
	}
	return emails, nil
}

// === Transaction ===

type MongoDBTransaction struct {
	*MongoDBProvider
	session mongo.Session
}

func (t *MongoDBTransaction) sessionCtx() context.Context {
	return mongo.NewSessionContext(context.TODO(), t.session)
}

func (p *MongoDBProvider) BeginTransaction() (DBProvider, error) {
	session, err := p.client.StartSession()
	if err != nil {
		return nil, err
	}
	if err := session.StartTransaction(); err != nil {
		session.EndSession(context.Background())
		return nil, err
	}
	return &MongoDBTransaction{MongoDBProvider: p, session: session}, nil
}

func (t *MongoDBTransaction) CommitTransaction() error {
	err := t.session.CommitTransaction(context.TODO())
	t.session.EndSession(context.Background())
	return err
}

func (t *MongoDBTransaction) RollbackTransaction() error {
	err := t.session.AbortTransaction(context.TODO())
	t.session.EndSession(context.Background())
	return err
}

// No-op methods on MongoDBProvider
func (p *MongoDBProvider) Checkpoint() error                         { return nil }
func (p *MongoDBProvider) IsSQLiteDB(file io.ReaderAt) (bool, error) { return false, nil }
func (p *MongoDBProvider) GetGormDB() *gorm.DB                      { return nil }
func (p *MongoDBProvider) CommitTransaction() error                  { return nil }
func (p *MongoDBProvider) RollbackTransaction() error                { return nil }

// No-op methods on MongoDBTransaction
func (t *MongoDBTransaction) Init(dbPath string) error                  { return nil }
func (t *MongoDBTransaction) Close() error                              { return nil }
func (t *MongoDBTransaction) Checkpoint() error                         { return nil }
func (t *MongoDBTransaction) IsSQLiteDB(file io.ReaderAt) (bool, error) { return false, nil }
func (t *MongoDBTransaction) GetGormDB() *gorm.DB                      { return nil }
func (t *MongoDBTransaction) IsNotFound(err error) bool                 { return err == mongo.ErrNoDocuments }
func (t *MongoDBTransaction) BeginTransaction() (DBProvider, error)     { return nil, nil }

// Transaction User methods

func (t *MongoDBTransaction) GetFirstUser() (*model.User, error) {
	user := &model.User{}
	err := t.db.Collection("users").FindOne(t.sessionCtx(), bson.M{}).Decode(user)
	return user, err
}

func (t *MongoDBTransaction) GetUserByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := t.db.Collection("users").FindOne(t.sessionCtx(), bson.M{"username": username}).Decode(user)
	return user, err
}

func (t *MongoDBTransaction) CreateUser(user *model.User) error {
	id, err := t.getNextSequence("users")
	if err != nil {
		return err
	}
	user.Id = id
	_, err = t.db.Collection("users").InsertOne(t.sessionCtx(), user)
	return err
}

func (t *MongoDBTransaction) UpdateUserByID(id int, updates map[string]any) error {
	_, err := t.db.Collection("users").UpdateOne(t.sessionCtx(), bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (t *MongoDBTransaction) SaveUser(user *model.User) error {
	_, err := t.db.Collection("users").ReplaceOne(t.sessionCtx(), bson.M{"_id": user.Id}, user)
	return err
}

func (t *MongoDBTransaction) DeleteUserByUsername(username string) error {
	_, err := t.db.Collection("users").DeleteOne(t.sessionCtx(), bson.M{"username": username})
	return err
}

func (t *MongoDBTransaction) GetAllUsers() ([]*model.User, error) {
	cursor, err := t.db.Collection("users").Find(t.sessionCtx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var users []*model.User
	if err := cursor.All(t.sessionCtx(), &users); err != nil {
		return nil, err
	}
	return users, nil
}

// Transaction Inbound methods

func (t *MongoDBTransaction) GetInboundsWithClientStats() ([]*model.Inbound, error) {
	cursor, err := t.db.Collection("inbounds").Find(t.sessionCtx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var inbounds []*model.Inbound
	if err := cursor.All(t.sessionCtx(), &inbounds); err != nil {
		return nil, err
	}
	return t.populateClientStats(inbounds)
}

func (t *MongoDBTransaction) GetAllInboundsWithClientStats() ([]*model.Inbound, error) {
	return t.GetInboundsWithClientStats()
}

func (t *MongoDBTransaction) GetInboundByID(id int) (*model.Inbound, error) {
	inbound := &model.Inbound{}
	err := t.db.Collection("inbounds").FindOne(t.sessionCtx(), bson.M{"_id": id}).Decode(inbound)
	return inbound, err
}

func (t *MongoDBTransaction) CreateInbound(inbound *model.Inbound) error {
	id, err := t.getNextSequence("inbounds")
	if err != nil {
		return err
	}
	inbound.Id = id
	_, err = t.db.Collection("inbounds").InsertOne(t.sessionCtx(), inbound)
	return err
}

func (t *MongoDBTransaction) SaveInbound(inbound *model.Inbound) error {
	_, err := t.db.Collection("inbounds").ReplaceOne(t.sessionCtx(), bson.M{"_id": inbound.Id}, inbound)
	return err
}

func (t *MongoDBTransaction) DeleteInboundByID(id int) error {
	_, err := t.db.Collection("inbounds").DeleteOne(t.sessionCtx(), bson.M{"_id": id})
	return err
}

func (t *MongoDBTransaction) GetInboundTagByID(id int) (string, error) {
	var result struct {
		Tag string `bson:"tag"`
	}
	err := t.db.Collection("inbounds").FindOne(t.sessionCtx(), bson.M{"_id": id}).Decode(&result)
	return result.Tag, err
}

func (t *MongoDBTransaction) GetInboundIDs() ([]int, error) {
	results, err := t.db.Collection("inbounds").Distinct(t.sessionCtx(), "_id", bson.M{})
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(results))
	for i, v := range results {
		switch val := v.(type) {
		case int32:
			ids[i] = int(val)
		case int64:
			ids[i] = int(val)
		case int:
			ids[i] = val
		default:
			ids[i] = 0
		}
	}
	return ids, nil
}

func (t *MongoDBTransaction) CountInboundsByPort(port int) (int64, error) {
	return t.db.Collection("inbounds").CountDocuments(t.sessionCtx(), bson.M{"port": port})
}

func (t *MongoDBTransaction) GetInboundEmails() ([]string, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"settings.clients": bson.M{"$exists": true}}}},
		{{Key: "$unwind", Value: "$settings.clients"}},
		{{Key: "$match", Value: bson.M{"settings.clients.email": bson.M{"$ne": ""}}}},
		{{Key: "$group", Value: bson.M{"_id": "$settings.clients.email"}}},
	}
	cursor, err := t.db.Collection("inbounds").Aggregate(t.sessionCtx(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var results []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(t.sessionCtx(), &results); err != nil {
		return nil, err
	}
	emails := make([]string, len(results))
	for i, r := range results {
		emails[i] = r.ID
	}
	return emails, nil
}

// === Client Traffic ===

func (p *MongoDBProvider) CreateClientTraffic(traffic *xray.ClientTraffic) error {
	id, err := p.getNextSequence("client_traffics")
	if err != nil {
		return err
	}
	traffic.Id = id
	_, err = p.db.Collection("client_traffics").InsertOne(context.TODO(), traffic)
	return err
}

func (p *MongoDBProvider) SaveClientTraffic(traffic *xray.ClientTraffic) error {
	_, err := p.db.Collection("client_traffics").ReplaceOne(context.TODO(), bson.M{"_id": traffic.Id}, traffic)
	return err
}

func (p *MongoDBProvider) GetClientTrafficsByEmails(emails []string) ([]*xray.ClientTraffic, error) {
	cursor, err := p.db.Collection("client_traffics").Find(context.TODO(), bson.M{"email": bson.M{"$in": emails}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var traffics []*xray.ClientTraffic
	if err := cursor.All(context.TODO(), &traffics); err != nil {
		return nil, err
	}
	return traffics, nil
}

func (p *MongoDBProvider) GetClientTrafficsByIDs(ids []int) ([]*xray.ClientTraffic, error) {
	cursor, err := p.db.Collection("client_traffics").Find(context.TODO(), bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var traffics []*xray.ClientTraffic
	if err := cursor.All(context.TODO(), &traffics); err != nil {
		return nil, err
	}
	return traffics, nil
}

func (p *MongoDBProvider) DeleteClientTrafficByID(id int) error {
	_, err := p.db.Collection("client_traffics").DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

func (p *MongoDBProvider) UpdateClientTrafficByEmail(email string, up, down int64) error {
	_, err := p.db.Collection("client_traffics").UpdateOne(context.TODO(),
		bson.M{"email": email},
		bson.M{"$inc": bson.M{"up": up, "down": down}},
	)
	return err
}

func (p *MongoDBProvider) UpdateClientTrafficsBatch(traffics []*xray.ClientTraffic) error {
	if len(traffics) == 0 {
		return nil
	}
	var models []mongo.WriteModel
	for _, t := range traffics {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.M{"email": t.Email}).
			SetReplacement(t))
	}
	_, err := p.db.Collection("client_traffics").BulkWrite(context.TODO(), models)
	return err
}

func (p *MongoDBProvider) SelectClientTrafficEnableByEmail(email string) (bool, error) {
	var result struct {
		Enable bool `bson:"enable"`
	}
	err := p.db.Collection("client_traffics").FindOne(context.TODO(), bson.M{"email": email}).Decode(&result)
	return result.Enable, err
}

// Transaction Client Traffic methods

func (t *MongoDBTransaction) CreateClientTraffic(traffic *xray.ClientTraffic) error {
	id, err := t.getNextSequence("client_traffics")
	if err != nil {
		return err
	}
	traffic.Id = id
	_, err = t.db.Collection("client_traffics").InsertOne(t.sessionCtx(), traffic)
	return err
}

func (t *MongoDBTransaction) SaveClientTraffic(traffic *xray.ClientTraffic) error {
	_, err := t.db.Collection("client_traffics").ReplaceOne(t.sessionCtx(), bson.M{"_id": traffic.Id}, traffic)
	return err
}

func (t *MongoDBTransaction) GetClientTrafficsByEmails(emails []string) ([]*xray.ClientTraffic, error) {
	cursor, err := t.db.Collection("client_traffics").Find(t.sessionCtx(), bson.M{"email": bson.M{"$in": emails}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var traffics []*xray.ClientTraffic
	if err := cursor.All(t.sessionCtx(), &traffics); err != nil {
		return nil, err
	}
	return traffics, nil
}

func (t *MongoDBTransaction) GetClientTrafficsByIDs(ids []int) ([]*xray.ClientTraffic, error) {
	cursor, err := t.db.Collection("client_traffics").Find(t.sessionCtx(), bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var traffics []*xray.ClientTraffic
	if err := cursor.All(t.sessionCtx(), &traffics); err != nil {
		return nil, err
	}
	return traffics, nil
}

func (t *MongoDBTransaction) DeleteClientTrafficByID(id int) error {
	_, err := t.db.Collection("client_traffics").DeleteOne(t.sessionCtx(), bson.M{"_id": id})
	return err
}

func (t *MongoDBTransaction) UpdateClientTrafficByEmail(email string, up, down int64) error {
	_, err := t.db.Collection("client_traffics").UpdateOne(t.sessionCtx(),
		bson.M{"email": email},
		bson.M{"$inc": bson.M{"up": up, "down": down}},
	)
	return err
}

func (t *MongoDBTransaction) UpdateClientTrafficsBatch(traffics []*xray.ClientTraffic) error {
	if len(traffics) == 0 {
		return nil
	}
	var models []mongo.WriteModel
	for _, traffic := range traffics {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.M{"email": traffic.Email}).
			SetReplacement(traffic))
	}
	_, err := t.db.Collection("client_traffics").BulkWrite(t.sessionCtx(), models)
	return err
}

func (t *MongoDBTransaction) SelectClientTrafficEnableByEmail(email string) (bool, error) {
	var result struct {
		Enable bool `bson:"enable"`
	}
	err := t.db.Collection("client_traffics").FindOne(t.sessionCtx(), bson.M{"email": email}).Decode(&result)
	return result.Enable, err
}

// === Client IPs ===

func (p *MongoDBProvider) GetAllInboundClientIps() ([]*model.InboundClientIps, error) {
	cursor, err := p.db.Collection("inbound_client_ips").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var ips []*model.InboundClientIps
	if err := cursor.All(context.TODO(), &ips); err != nil {
		return nil, err
	}
	return ips, nil
}

func (p *MongoDBProvider) GetInboundClientIpsByEmail(clientEmail string) (*model.InboundClientIps, error) {
	ip := &model.InboundClientIps{}
	err := p.db.Collection("inbound_client_ips").FindOne(context.TODO(), bson.M{"client_email": clientEmail}).Decode(ip)
	return ip, err
}

func (p *MongoDBProvider) SaveInboundClientIps(ips *model.InboundClientIps) error {
	filter := bson.M{"client_email": ips.ClientEmail}
	_, err := p.db.Collection("inbound_client_ips").ReplaceOne(context.TODO(), filter, ips, options.Replace().SetUpsert(true))
	return err
}

func (p *MongoDBProvider) DeleteClientIpsByEmail(clientEmail string) error {
	_, err := p.db.Collection("inbound_client_ips").DeleteOne(context.TODO(), bson.M{"client_email": clientEmail})
	return err
}

func (p *MongoDBProvider) ClearClientIpsByEmail(clientEmail string) error {
	_, err := p.db.Collection("inbound_client_ips").UpdateOne(context.TODO(), bson.M{"client_email": clientEmail}, bson.M{"$set": bson.M{"ips": ""}})
	return err
}

// Transaction Client IPs methods

func (t *MongoDBTransaction) GetAllInboundClientIps() ([]*model.InboundClientIps, error) {
	cursor, err := t.db.Collection("inbound_client_ips").Find(t.sessionCtx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var ips []*model.InboundClientIps
	if err := cursor.All(t.sessionCtx(), &ips); err != nil {
		return nil, err
	}
	return ips, nil
}

func (t *MongoDBTransaction) GetInboundClientIpsByEmail(clientEmail string) (*model.InboundClientIps, error) {
	ip := &model.InboundClientIps{}
	err := t.db.Collection("inbound_client_ips").FindOne(t.sessionCtx(), bson.M{"client_email": clientEmail}).Decode(ip)
	return ip, err
}

func (t *MongoDBTransaction) SaveInboundClientIps(ips *model.InboundClientIps) error {
	filter := bson.M{"client_email": ips.ClientEmail}
	_, err := t.db.Collection("inbound_client_ips").ReplaceOne(t.sessionCtx(), filter, ips, options.Replace().SetUpsert(true))
	return err
}

func (t *MongoDBTransaction) DeleteClientIpsByEmail(clientEmail string) error {
	_, err := t.db.Collection("inbound_client_ips").DeleteOne(t.sessionCtx(), bson.M{"client_email": clientEmail})
	return err
}

func (t *MongoDBTransaction) ClearClientIpsByEmail(clientEmail string) error {
	_, err := t.db.Collection("inbound_client_ips").UpdateOne(t.sessionCtx(), bson.M{"client_email": clientEmail}, bson.M{"$set": bson.M{"ips": ""}})
	return err
}

// === Outbound Traffic ===

func (p *MongoDBProvider) GetOutboundTraffics() ([]*model.OutboundTraffics, error) {
	cursor, err := p.db.Collection("outbound_traffics").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var traffics []*model.OutboundTraffics
	if err := cursor.All(context.TODO(), &traffics); err != nil {
		return nil, err
	}
	return traffics, nil
}

func (p *MongoDBProvider) FirstOrCreateOutboundTraffic(tag string) (*model.OutboundTraffics, error) {
	traffic := &model.OutboundTraffics{}
	err := p.db.Collection("outbound_traffics").FindOne(context.TODO(), bson.M{"tag": tag}).Decode(traffic)
	if err == mongo.ErrNoDocuments {
		id, err := p.getNextSequence("outbound_traffics")
		if err != nil {
			return nil, err
		}
		traffic.Id = id
		traffic.Tag = tag
		_, err = p.db.Collection("outbound_traffics").InsertOne(context.TODO(), traffic)
		return traffic, err
	}
	return traffic, err
}

func (p *MongoDBProvider) SaveOutboundTraffic(traffic *model.OutboundTraffics) error {
	_, err := p.db.Collection("outbound_traffics").ReplaceOne(context.TODO(), bson.M{"_id": traffic.Id}, traffic)
	return err
}

func (p *MongoDBProvider) ResetOutboundTraffics(tag string, allTags bool) error {
	filter := bson.M{}
	if !allTags {
		filter = bson.M{"tag": tag}
	}
	_, err := p.db.Collection("outbound_traffics").UpdateMany(context.TODO(), filter, bson.M{"$set": bson.M{
		"up":    0,
		"down":  0,
		"total": 0,
	}})
	return err
}

// Transaction Outbound Traffic methods

func (t *MongoDBTransaction) GetOutboundTraffics() ([]*model.OutboundTraffics, error) {
	cursor, err := t.db.Collection("outbound_traffics").Find(t.sessionCtx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var traffics []*model.OutboundTraffics
	if err := cursor.All(t.sessionCtx(), &traffics); err != nil {
		return nil, err
	}
	return traffics, nil
}

func (t *MongoDBTransaction) FirstOrCreateOutboundTraffic(tag string) (*model.OutboundTraffics, error) {
	traffic := &model.OutboundTraffics{}
	err := t.db.Collection("outbound_traffics").FindOne(t.sessionCtx(), bson.M{"tag": tag}).Decode(traffic)
	if err == mongo.ErrNoDocuments {
		id, err := t.getNextSequence("outbound_traffics")
		if err != nil {
			return nil, err
		}
		traffic.Id = id
		traffic.Tag = tag
		_, err = t.db.Collection("outbound_traffics").InsertOne(t.sessionCtx(), traffic)
		return traffic, err
	}
	return traffic, err
}

func (t *MongoDBTransaction) SaveOutboundTraffic(traffic *model.OutboundTraffics) error {
	_, err := t.db.Collection("outbound_traffics").ReplaceOne(t.sessionCtx(), bson.M{"_id": traffic.Id}, traffic)
	return err
}

func (t *MongoDBTransaction) ResetOutboundTraffics(tag string, allTags bool) error {
	filter := bson.M{}
	if !allTags {
		filter = bson.M{"tag": tag}
	}
	_, err := t.db.Collection("outbound_traffics").UpdateMany(t.sessionCtx(), filter, bson.M{"$set": bson.M{
		"up":    0,
		"down":  0,
		"total": 0,
	}})
	return err
}

// === Settings ===

func (p *MongoDBProvider) GetAllSettings() ([]*model.Setting, error) {
	cursor, err := p.db.Collection("settings").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var settings []*model.Setting
	if err := cursor.All(context.TODO(), &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (p *MongoDBProvider) GetSettingByKey(key string) (*model.Setting, error) {
	setting := &model.Setting{}
	err := p.db.Collection("settings").FindOne(context.TODO(), bson.M{"key": key}).Decode(setting)
	return setting, err
}

func (p *MongoDBProvider) CreateSetting(setting *model.Setting) error {
	id, err := p.getNextSequence("settings")
	if err != nil {
		return err
	}
	setting.Id = id
	_, err = p.db.Collection("settings").InsertOne(context.TODO(), setting)
	return err
}

func (p *MongoDBProvider) SaveSetting(setting *model.Setting) error {
	_, err := p.db.Collection("settings").ReplaceOne(context.TODO(), bson.M{"_id": setting.Id}, setting)
	return err
}

func (p *MongoDBProvider) DeleteAllSettings() error {
	_, err := p.db.Collection("settings").DeleteMany(context.TODO(), bson.M{})
	return err
}

// Transaction Settings methods

func (t *MongoDBTransaction) GetAllSettings() ([]*model.Setting, error) {
	cursor, err := t.db.Collection("settings").Find(t.sessionCtx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var settings []*model.Setting
	if err := cursor.All(t.sessionCtx(), &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (t *MongoDBTransaction) GetSettingByKey(key string) (*model.Setting, error) {
	setting := &model.Setting{}
	err := t.db.Collection("settings").FindOne(t.sessionCtx(), bson.M{"key": key}).Decode(setting)
	return setting, err
}

func (t *MongoDBTransaction) CreateSetting(setting *model.Setting) error {
	id, err := t.getNextSequence("settings")
	if err != nil {
		return err
	}
	setting.Id = id
	_, err = t.db.Collection("settings").InsertOne(t.sessionCtx(), setting)
	return err
}

func (t *MongoDBTransaction) SaveSetting(setting *model.Setting) error {
	_, err := t.db.Collection("settings").ReplaceOne(t.sessionCtx(), bson.M{"_id": setting.Id}, setting)
	return err
}

func (t *MongoDBTransaction) DeleteAllSettings() error {
	_, err := t.db.Collection("settings").DeleteMany(t.sessionCtx(), bson.M{})
	return err
}

// === Link History ===

func (p *MongoDBProvider) AddLinkHistory(record *LinkHistory) error {
	id, err := p.getNextSequence("link_histories")
	if err != nil {
		return err
	}
	record.Id = id
	_, err = p.db.Collection("link_histories").InsertOne(context.TODO(), record)
	return err
}

func (p *MongoDBProvider) GetLinkHistory() ([]*LinkHistory, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(10)
	cursor, err := p.db.Collection("link_histories").Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var history []*LinkHistory
	if err := cursor.All(context.TODO(), &history); err != nil {
		return nil, err
	}
	return history, nil
}

// Transaction Link History methods

func (t *MongoDBTransaction) AddLinkHistory(record *LinkHistory) error {
	id, err := t.getNextSequence("link_histories")
	if err != nil {
		return err
	}
	record.Id = id
	_, err = t.db.Collection("link_histories").InsertOne(t.sessionCtx(), record)
	return err
}

func (t *MongoDBTransaction) GetLinkHistory() ([]*LinkHistory, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(10)
	cursor, err := t.db.Collection("link_histories").Find(t.sessionCtx(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var history []*LinkHistory
	if err := cursor.All(t.sessionCtx(), &history); err != nil {
		return nil, err
	}
	return history, nil
}

// === Lottery ===

func (p *MongoDBProvider) HasUserWonToday(userID int64) (bool, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	count, err := p.db.Collection("lottery_wins").CountDocuments(context.TODO(), bson.M{
		"user_id":  userID,
		"win_date": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p *MongoDBProvider) RecordUserWin(userID int64, prize string) error {
	id, err := p.getNextSequence("lottery_wins")
	if err != nil {
		return err
	}
	winRecord := &model.LotteryWin{
		ID:      int64(id),
		UserID:  userID,
		Prize:   prize,
		WinDate: time.Now(),
	}
	_, err = p.db.Collection("lottery_wins").InsertOne(context.TODO(), winRecord)
	return err
}

// Transaction Lottery methods

func (t *MongoDBTransaction) HasUserWonToday(userID int64) (bool, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	count, err := t.db.Collection("lottery_wins").CountDocuments(t.sessionCtx(), bson.M{
		"user_id":  userID,
		"win_date": bson.M{"$gte": startOfDay, "$lt": endOfDay},
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (t *MongoDBTransaction) RecordUserWin(userID int64, prize string) error {
	id, err := t.getNextSequence("lottery_wins")
	if err != nil {
		return err
	}
	winRecord := &model.LotteryWin{
		ID:      int64(id),
		UserID:  userID,
		Prize:   prize,
		WinDate: time.Now(),
	}
	_, err = t.db.Collection("lottery_wins").InsertOne(t.sessionCtx(), winRecord)
	return err
}

// === Seeder ===

func (p *MongoDBProvider) GetSeederNames() ([]string, error) {
	cursor, err := p.db.Collection("history_of_seeders").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []struct {
		SeederName string `bson:"seeder_name"`
	}
	if err := cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.SeederName
	}
	return names, nil
}

func (p *MongoDBProvider) CreateSeederHistory(name string) error {
	id, err := p.getNextSequence("history_of_seeders")
	if err != nil {
		return err
	}
	record := &model.HistoryOfSeeders{
		Id:         id,
		SeederName: name,
	}
	_, err = p.db.Collection("history_of_seeders").InsertOne(context.TODO(), record)
	return err
}

func (p *MongoDBProvider) IsTableEmpty(tableName string) (bool, error) {
	count, err := p.db.Collection(tableName).CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// Transaction Seeder methods

func (t *MongoDBTransaction) GetSeederNames() ([]string, error) {
	cursor, err := t.db.Collection("history_of_seeders").Find(t.sessionCtx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(t.sessionCtx())

	var results []struct {
		SeederName string `bson:"seeder_name"`
	}
	if err := cursor.All(t.sessionCtx(), &results); err != nil {
		return nil, err
	}
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.SeederName
	}
	return names, nil
}

func (t *MongoDBTransaction) CreateSeederHistory(name string) error {
	id, err := t.getNextSequence("history_of_seeders")
	if err != nil {
		return err
	}
	record := &model.HistoryOfSeeders{
		Id:         id,
		SeederName: name,
	}
	_, err = t.db.Collection("history_of_seeders").InsertOne(t.sessionCtx(), record)
	return err
}

func (t *MongoDBTransaction) IsTableEmpty(tableName string) (bool, error) {
	count, err := t.db.Collection(tableName).CountDocuments(t.sessionCtx(), bson.M{})
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// === Advanced inbound ===

func (p *MongoDBProvider) DisableInvalidInbounds(expiryTime int64) (int64, error) {
	filter := bson.M{"enable": true, "expiry_time": bson.M{"$gt": 0, "$lt": expiryTime}}
	update := bson.M{"$set": bson.M{"enable": false}}
	result, err := p.db.Collection("inbounds").UpdateMany(context.TODO(), filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (p *MongoDBProvider) DisableInvalidClients(expiryTime int64) (int64, error) {
	filter := bson.M{"enable": true, "expiry_time": bson.M{"$gt": 0, "$lt": expiryTime}}
	update := bson.M{"$set": bson.M{"enable": false}}
	result, err := p.db.Collection("client_traffics").UpdateMany(context.TODO(), filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (p *MongoDBProvider) MigrationRemoveOrphanedTraffics() error {
	// Get all valid emails from inbounds
	emails, err := p.GetInboundEmails()
	if err != nil {
		return err
	}
	if len(emails) == 0 {
		return nil
	}
	_, err = p.db.Collection("client_traffics").DeleteMany(context.TODO(), bson.M{
		"email": bson.M{"$nin": emails},
	})
	return err
}

// Transaction Advanced inbound methods

func (t *MongoDBTransaction) DisableInvalidInbounds(expiryTime int64) (int64, error) {
	filter := bson.M{"enable": true, "expiry_time": bson.M{"$gt": 0, "$lt": expiryTime}}
	update := bson.M{"$set": bson.M{"enable": false}}
	result, err := t.db.Collection("inbounds").UpdateMany(t.sessionCtx(), filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (t *MongoDBTransaction) DisableInvalidClients(expiryTime int64) (int64, error) {
	filter := bson.M{"enable": true, "expiry_time": bson.M{"$gt": 0, "$lt": expiryTime}}
	update := bson.M{"$set": bson.M{"enable": false}}
	result, err := t.db.Collection("client_traffics").UpdateMany(t.sessionCtx(), filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (t *MongoDBTransaction) MigrationRemoveOrphanedTraffics() error {
	emails, err := t.GetInboundEmails()
	if err != nil {
		return err
	}
	if len(emails) == 0 {
		return nil
	}
	_, err = t.db.Collection("client_traffics").DeleteMany(t.sessionCtx(), bson.M{
		"email": bson.M{"$nin": emails},
	})
	return err
}
