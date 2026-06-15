package mongodb_test

import (
	"context"
	"fmt"
	"github.com/name5566/leaf/db/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Example() {
	// In a real environment, use a proper MongoDB URI
	c, err := mongodb.Dial("mongodb://localhost:27017", 10)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	// session
	s := c.Ref()
	defer c.UnRef(s)
	
	ctx := context.Background()
	_, err = s.Database("test").Collection("counters").DeleteOne(ctx, bson.M{"_id": "test"})
	if err != nil && err != mongo.ErrNoDocuments {
		// Note: DeleteOne doesn't return ErrNoDocuments if no doc matches, it returns Result.DeletedCount = 0
		// But let's keep it simple for example.
	}

	// auto increment
	err = c.EnsureCounter("test", "counters", "test")
	if err != nil {
		fmt.Println(err)
		return
	}
	for i := 0; i < 3; i++ {
		id, err := c.NextSeq("test", "counters", "test")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(id)
	}

	// index
	c.EnsureUniqueIndex("test", "counters", []string{"key1"})

	// Output:
	// 1
	// 2
	// 3
}
