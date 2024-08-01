package news

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
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

func fetchDecodedBatchExecute(id string) (string, error) {
	s := `[[["Fbv4je","[\"garturlreq\",[[\"en-US\",\"US\",[\"FINANCE_TOP_INDICES\",\"WEB_TEST_1_0_0\"],` +
		`null,null,1,1,\"US:en\",null,180,null,null,null,null,null,0,null,null,[1608992183,723341000]],` +
		`\"en-US\",\"US\",1,[2,3,4,8],1,0,\"655000234\",0,0,null,0],\"` + id + `\"]",null,"generic"]]]`

	data := url.Values{}
	data.Set("f.req", s)

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://news.google.com/_/DotsSplashUi/data/batchexecute?rpcids=Fbv4je", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	req.Header.Add("Referer", "https://news.google.com/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch data from Google")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	text := string(body)
	header := `[\"garturlres\",\"`
	footer := `\",`

	if !strings.Contains(text, header) {
		return "", fmt.Errorf("header not found in response: %s", text)
	}

	parts := strings.SplitN(text, header, 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("header not found in response")
	}

	start := parts[1]
	if !strings.Contains(start, footer) {
		return "", fmt.Errorf("footer not found in response")
	}

	urlParts := strings.SplitN(start, footer, 2)
	if len(urlParts) < 2 {
		return "", fmt.Errorf("URL not found in response")
	}

	return urlParts[0], nil
}

func decodeGoogleNewsURL(sourceURL string) (string, error) {
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		return "", err
	}

	path := strings.Split(parsedURL.Path, "/")
	if parsedURL.Hostname() == "news.google.com" && len(path) > 1 && path[len(path)-2] == "articles" {
		base64Str := path[len(path)-1]

		// 移除可能的 URL 查询参数
		base64Str = strings.Split(base64Str, "?")[0]

		// 调整 base64 字符串长度为 4 的倍数
		if len(base64Str)%4 != 0 {
			base64Str += strings.Repeat("=", 4-len(base64Str)%4)
		}

		decodedBytes, err := base64.URLEncoding.DecodeString(base64Str)
		if err != nil {
			return "", fmt.Errorf("base64 decode error: %v", err)
		}

		decodedStr := string(decodedBytes)
		prefix := "\x08\x13\x22"
		suffix := "\xd2\x01\x00"

		if strings.HasPrefix(decodedStr, prefix) {
			decodedStr = decodedStr[len(prefix):]
		}
		if strings.HasSuffix(decodedStr, suffix) {
			decodedStr = decodedStr[:len(decodedStr)-len(suffix)]
		}

		bytesArray := []byte(decodedStr)
		if len(bytesArray) == 0 {
			return "", fmt.Errorf("decoded string is empty")
		}
		length := int(bytesArray[0])
		if length >= 0x80 {
			if len(bytesArray) < 2 {
				return "", fmt.Errorf("decoded string is too short")
			}
			decodedStr = decodedStr[2 : length+1]
		} else {
			decodedStr = decodedStr[1 : length+1]
		}

		if strings.HasPrefix(decodedStr, "AU_yqL") {
			return fetchDecodedBatchExecute(base64Str)
		}
		return decodedStr, nil
	}
	return sourceURL, nil
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

		tmpLink, err := decodeGoogleNewsURL(news.Link)

		if err != nil {
			news.SourceLink = news.Link
		} else {
			news.SourceLink = tmpLink
		}

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

func FixedData(topic string) {
	client := db.Connect()
	defer db.Disconnect(client)
	coll := client.Database("test").Collection(topic)
	// 查询集合中的所有文档
	cursor, err := coll.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var item NewsItem
		err := cursor.Decode(&item)
		if err != nil {
			log.Fatal(err)
		}

		// 检查 link 属性是否为空
		if item.Link == "" {
			// 更新文档

			filter := bson.M{"id": item.ID}
			update := bson.M{"$set": bson.M{"link": "default_link"}} // 将 link 更新为默认值 "default_link"
			_, err := coll.UpdateOne(context.TODO(), filter, update)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Updated document with id: %v\n", item.ID)
		}
	}

	if err := cursor.Err(); err != nil {
		log.Fatal(err)
	}

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
