package news

import (
	"context"
	"log"
	"time"

	"github.com/Zhima-Mochi/newsApi-go/newsapi"
	ai "github.com/mentaLwz/tesla-news-parse/ai"
	db "github.com/mentaLwz/tesla-news-parse/db"
	lru "github.com/mentaLwz/tesla-news-parse/lru"
	"go.mongodb.org/mongo-driver/bson"
)

type NewsItem struct {
	ID     int64      `bson:"id"`
	Title  string     `bson:"title"`
	Date   *time.Time `bson:"date"`
	Link   string     `bson:"link"`
	Source string     `bson:"source,omitempty"`
	Score  string     `bson:"score"`
	guid   string
}

func CollectNews(topic string, durationHour int) []NewsItem {
	handler := newsapi.NewNewsApi()
	queryOptions := []newsapi.QueryOption{
		newsapi.WithLanguage(newsapi.LanguageEnglish),
		newsapi.WithLocation(newsapi.LocationUnitedStates),
		newsapi.WithLimit(30),
		newsapi.WithStartDate(time.Now().Add(-time.Hour * time.Duration(durationHour*72))),
		newsapi.WithEndDate(time.Now()),
	}
	handler.SetQueryOptions(queryOptions...)

	newsList, err := handler.SearchNews(topic)
	if err != nil {
		log.Println(err)
		return nil
	}
	newsapi.FetchSourceLinks(newsList)

	var newsCollections []NewsItem
	for _, news := range newsList {
		newsapi.FetchSourceContents([]*newsapi.News{news})

		if true {
			log.Println("==========debug=============")
			log.Println("SourceLink", news.SourceLink)
			log.Println("SourceTitle", news.SourceTitle)
			log.Println("SourceImageURL", news.SourceImageURL)
			log.Println("SourceImageWidth", news.SourceImageWidth)
			log.Println("SourceImageHeight", news.SourceImageHeight)
			log.Println("SourceDescription", news.SourceDescription)
			log.Println("SourceKeywords", news.SourceKeywords)
			log.Println("SourceSiteName", news.SourceSiteName)
			log.Println("Published", news.Published)
		}

		newsCollections = append(newsCollections, NewsItem{
			Title:  news.Title,
			Date:   news.PublishedParsed,
			Link:   news.SourceLink,
			Source: news.SourceSiteName,
			guid:   news.GUID,
		})
	}

	log.Printf("Collected %d news items", len(newsCollections))
	return newsCollections
}

func InsertData(topic string, data []NewsItem) {
	client := db.Connect()
	defer db.Disconnect(client)

	cache := lru.GetInstance(1000)
	// Send a ping to confirm a successful connection
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}
	counterName := topic + "-counter"
	coll := client.Database("test").Collection("counters")
	filter := bson.D{{"name", counterName}}

	newData := []interface{}{}
	for i := range data {
		score := ai.GetScore(data[i].Title)
		if cache.Contains(data[i].guid) {
			continue
		} else {
			cache.Put(data[i].guid)
		}

		if score == "0" {
			continue
		}
		data[i].Score = score
		newData = append(newData, data[i])
	}
	len := len(newData)

	if len == 0 {
		println("no update this time")
		return
	}

	update := bson.D{{"$inc", bson.D{{"seq", len}}}}

	var prevDoc db.Counter

	err := coll.FindOneAndUpdate(context.TODO(), filter, update).Decode(&prevDoc)
	if err != nil {
		panic(err)
	}
	seq := prevDoc.Seq

	for i := range newData {
		newsItem := newData[i].(NewsItem)
		newsItem.ID = seq + 1
		seq += 1
		newData[i] = newsItem
	}

	println("Updated seq", seq)

	coll = client.Database("test").Collection(topic)
	result, err := coll.InsertMany(context.TODO(), newData)
	println("Updated result", result)
	println(result)
	if err != nil {
		panic(err)
	}
}
