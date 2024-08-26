package news

import (
	"context"
	"fmt"
	"testing"

	"github.com/mentaLwz/tesla-news-parse/config"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Counter struct {
	Name string `bson:"name"`
	Seq  int64  `bson:"seq"`
}

func createCounter(databseName string) {

	out := viper.GetString("API_KEY") // case-insensitive Setting & Getting
	fmt.Println(out)

	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(out).SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	// Send a ping to confirm a successful connection
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	// client.Database("test").CreateCollection(context.TODO(), "counters")
	coll := client.Database(databseName).Collection("counters")
	newCounter := Counter{Name: "tesla-counter", Seq: 0}
	result, err := coll.InsertOne(context.TODO(), newCounter)
	if err != nil {
		panic(err)
	}
	println(result.InsertedID)

}

// func TestCreateCounter(t *testing.T) {

// 	config.LoadConfig()
// 	// 设置配置
// 	createCounter("test2")
// }

func TestFullContentAnalyse(t *testing.T) {

	config.LoadConfig()
	// 设置配置
	topic := viper.GetString("topic")
	// duration := viper.GetInt("NEWS_DURATION_HOUR")
	newsItems := CollectNews(topic, 10)
	InsertDataEx(topic, "test2", newsItems)
}

func TestUpdateContentAnalyse(t *testing.T) {

	config.LoadConfig()
	// 设置配置
	topic := viper.GetString("topic")
	// duration := viper.GetInt("NEWS_DURATION_HOUR")
	UpdateAnalyseFieldInDatabase(topic)
}
