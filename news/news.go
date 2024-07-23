package news

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

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

func getOriginUrl(googleURL string) string {

	// 解析URL
	parsedURL, err := url.Parse(googleURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return ""
	}

	// 从路径中提取编码的部分
	parts := strings.Split(parsedURL.Path, "/")
	if len(parts) < 3 {
		fmt.Println("Invalid URL format")
		return ""
	}
	encodedPart := parts[len(parts)-1] // 取最后一个部分
	fmt.Println("Encoded part:", encodedPart)

	// 解码Base64URL
	decodedBytes, err := base64.RawURLEncoding.DecodeString(encodedPart)
	if err != nil {
		fmt.Println("Error decoding base64:", err)
		return ""
	}
	decodedURL := string(decodedBytes)
	fmt.Println("Decoded URL:", decodedURL)

	// 清理解码后的URL
	originalURL := cleanURL(decodedURL)
	fmt.Println("原始链接:", originalURL)

	// 发送HTTP请求以处理可能的重定向
	resp, err := http.Get(originalURL)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return ""
	}
	defer resp.Body.Close()

	fmt.Println("最终链接:", resp.Request.URL.String())
	return resp.Request.URL.String()
}

func findAllIndex(s, substr string) []int {
	var indices []int
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], substr)
		if j == -1 {
			break
		}
		indices = append(indices, i+j)
		i += j + 1
	}
	return indices
}

func cleanURL(input string) string {
	// 查找所有 "http" 的位置
	httpIndices := findAllIndex(input, "http")

	if len(httpIndices) == 0 {
		return ""
	}

	// 选择第一个 "http" 位置（非AMP版本）
	startIndex := httpIndices[0]

	// 从选定的 "http" 开始截取字符串
	input = input[startIndex:]

	// 移除所有非打印字符和空白字符
	input = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return r
		}
		return -1
	}, input)
	input = strings.TrimSuffix(input, "�")
	return input
}

func CollectNews(topic string, durationHour int) []NewsItem {
	handler := newsapi.NewNewsApi()
	queryOptions := []newsapi.QueryOption{
		newsapi.WithLanguage(newsapi.LanguageEnglish),
		newsapi.WithLocation(newsapi.LocationUnitedStates),
		newsapi.WithLimit(30),
		newsapi.WithStartDate(time.Now().Add(-time.Hour * time.Duration(durationHour))),
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

		news.SourceLink = getOriginUrl(news.Link)
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
