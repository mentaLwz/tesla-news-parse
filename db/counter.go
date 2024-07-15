package db

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Counter struct {
	Name string `bson:"name"`
	Seq  int64  `bson:"seq"`
}

func CreateCounter(client *mongo.Client, name string) {
	coll := client.Database("test").Collection("counters")
	newCounter := Counter{Name: name, Seq: 0}
	_, err := coll.InsertOne(context.TODO(), newCounter)
	if err != nil {
		log.Fatal(err)
	}
}

func FindAndUpdateCounter(client *mongo.Client, name string, increment int) (Counter, error) {
	coll := client.Database("test").Collection("counters")
	filter := bson.D{{"name", name}}
	update := bson.D{{"$inc", bson.D{{"seq", increment}}}}

	var counter Counter
	err := coll.FindOneAndUpdate(context.TODO(), filter, update).Decode(&counter)
	return counter, err
}
