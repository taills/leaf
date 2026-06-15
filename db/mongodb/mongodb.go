package mongodb

import (
	"context"
	"github.com/name5566/leaf/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

// session
type Session struct {
	*mongo.Client
	ref int
}

type DialContext struct {
	sync.Mutex
	client *mongo.Client
	ref    int
}

// goroutine safe
func Dial(url string, sessionNum int) (*DialContext, error) {
	c, err := DialWithTimeout(url, sessionNum, 10*time.Second, 5*time.Minute)
	return c, err
}

// goroutine safe
func DialWithTimeout(url string, sessionNum int, dialTimeout time.Duration, timeout time.Duration) (*DialContext, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(url)
	clientOptions.SetConnectTimeout(dialTimeout)
	clientOptions.SetSocketTimeout(timeout)
	// sessionNum is ignored as mongo-driver handles connection pooling automatically.
	// We can set MaxPoolSize if needed.
	if sessionNum > 0 {
		clientOptions.SetMaxPoolSize(uint64(sessionNum))
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	c := new(DialContext)
	c.client = client

	return c, nil
}

// goroutine safe
func (c *DialContext) Close() {
	c.Lock()
	if c.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.client.Disconnect(ctx)
		if c.ref != 0 {
			log.Error("session ref = %v", c.ref)
		}
		c.client = nil
	}
	c.Unlock()
}

// goroutine safe
func (c *DialContext) Ref() *Session {
	c.Lock()
	c.ref++
	client := c.client
	c.Unlock()

	return &Session{Client: client}
}

// goroutine safe
func (c *DialContext) UnRef(s *Session) {
	c.Lock()
	c.ref--
	c.Unlock()
}

func IsDup(err error) bool {
	if mongo.IsDuplicateKeyError(err) {
		return true
	}
	return false
}

// goroutine safe
func (c *DialContext) EnsureCounter(db string, collection string, id string) error {
	s := c.Ref()
	defer c.UnRef(s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.Database(db).Collection(collection).InsertOne(ctx, bson.M{
		"_id": id,
		"seq": 0,
	})
	if IsDup(err) {
		return nil
	} else {
		return err
	}
}

// goroutine safe
func (c *DialContext) NextSeq(db string, collection string, id string) (int, error) {
	s := c.Ref()
	defer c.UnRef(s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var res struct {
		Seq int `bson:"seq"`
	}
	filter := bson.M{"_id": id}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := s.Database(db).Collection(collection).FindOneAndUpdate(ctx, filter, update, opts).Decode(&res)

	return res.Seq, err
}

// goroutine safe
func (c *DialContext) EnsureIndex(db string, collection string, key []string) error {
	s := c.Ref()
	defer c.UnRef(s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys := bson.D{}
	for _, k := range key {
		keys = append(keys, bson.E{Key: k, Value: 1})
	}

	indexModel := mongo.IndexModel{
		Keys: keys,
		Options: options.Index().SetSparse(true),
	}

	_, err := s.Database(db).Collection(collection).Indexes().CreateOne(ctx, indexModel)
	return err
}

// goroutine safe
func (c *DialContext) EnsureUniqueIndex(db string, collection string, key []string) error {
	s := c.Ref()
	defer c.UnRef(s)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys := bson.D{}
	for _, k := range key {
		keys = append(keys, bson.E{Key: k, Value: 1})
	}

	indexModel := mongo.IndexModel{
		Keys: keys,
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	_, err := s.Database(db).Collection(collection).Indexes().CreateOne(ctx, indexModel)
	return err
}
