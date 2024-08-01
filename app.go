package main

import (
	"bytes"
	"container/list"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/Zhima-Mochi/newsApi-go/newsapi"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Counter struct {
	Name string `bson:"name"`
	Seq  int64  `bson:"seq"`
}

type NewsItem struct {
	ID     int64      `bson:"id"`
	Title  string     `bson:"title"`
	Date   *time.Time `bson:"date"`
	Link   string     `bson:"link"`
	Source string     `bson:"source,omitempty"`
	Score  string     `bson:"score"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestData struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Response struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type LRUCache struct {
	capacity int
	cache    map[int]*list.Element
	list     *list.List
}

// 键值对结构体
type entry struct {
	key int
}

// NewLRUCache 创建一个新的 LRUCache
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[int]*list.Element),
		list:     list.New(),
	}
}

// Contains 检查缓存中是否包含某个键
func (l *LRUCache) Contains(key int) bool {
	if ele, found := l.cache[key]; found {
		l.list.MoveToFront(ele)
		return true
	}
	return false
}

// Put 将键放入缓存
func (l *LRUCache) Put(key int) {
	if ele, found := l.cache[key]; found {
		l.list.MoveToFront(ele)
		return
	}

	if l.list.Len() == l.capacity {
		back := l.list.Back()
		if back != nil {
			l.list.Remove(back)
			delete(l.cache, back.Value.(*entry).key)
		}
	}

	ele := l.list.PushFront(&entry{key})
	l.cache[key] = ele
}

func createCounter() {

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
	coll := client.Database("test").Collection("counters")
	newCounter := Counter{Name: "tesla-counter", Seq: 0}
	result, err := coll.InsertOne(context.TODO(), newCounter)
	if err != nil {
		panic(err)
	}
	println(result.InsertedID)

}

func createDB(dbName string) {

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
	if err := client.Database("test").CreateCollection(context.TODO(), dbName); err != nil {
		panic(err)
	}

	println("success create collections", dbName)

}

func testQuery() {
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
	// topic := "tesla"
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	// couterName := topic + "-counter"
	coll := client.Database("test").Collection("counters")
	filter := bson.D{{"name", "tesla-counter"}}

	var tc Counter
	err = coll.FindOne(context.TODO(), filter).Decode(&tc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No documents found")
		} else {
			panic(err)
		}
	}
	res, _ := bson.MarshalExtJSON(tc, false, false)
	fmt.Println(string(res))
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

func collectNews(topic string, durationHour int) []NewsItem {

	handler := newsapi.NewNewsApi()
	queryOptions := []newsapi.QueryOption{}
	queryOptions = append(queryOptions, newsapi.WithLanguage(newsapi.LanguageEnglish))
	queryOptions = append(queryOptions, newsapi.WithLocation(newsapi.LocationUnitedStates))
	queryOptions = append(queryOptions, newsapi.WithLimit(30))
	endDate := time.Now()
	startDate := endDate.Add(-time.Hour * time.Duration(durationHour))
	queryOptions = append(queryOptions, newsapi.WithStartDate(startDate))
	queryOptions = append(queryOptions, newsapi.WithEndDate(endDate))
	// queryOptions = append(queryOptions, newsapi.WithPeriod(time.Hour))
	handler.SetQueryOptions(queryOptions...)

	newsList, err := handler.SearchNews(topic)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	newsapi.FetchSourceLinks(newsList)

	newsCollections := []NewsItem{}

	for _, news := range newsList {
		fmt.Println("=================================")
		fmt.Println(news.Title)
		fmt.Println(news.Link)
		fmt.Println(news.SourceLink)
		news.SourceLink = getOriginUrl(news.Link)

		newsapi.FetchSourceContents([]*newsapi.News{news})

		if true {
			fmt.Println("SourceLink", news.SourceLink)
			fmt.Println("SourceTitle", news.SourceTitle)
			fmt.Println("SourceImageURL", news.SourceImageURL)
			fmt.Println("SourceImageWidth", news.SourceImageWidth)
			fmt.Println("SourceImageHeight", news.SourceImageHeight)
			fmt.Println("SourceDescription", news.SourceDescription)
			fmt.Println("SourceKeywords", news.SourceKeywords)
			fmt.Println("SourceSiteName", news.SourceSiteName)
			fmt.Println("Published", news.Published)
		}
		newsCollections = append(newsCollections, NewsItem{
			Title:  news.Title,
			Date:   news.PublishedParsed,
			Link:   news.SourceLink,
			Source: news.SourceSiteName,
		})

	}
	println("collect %d news", len(newsCollections))
	return newsCollections

}

func insertData(topic string, data []NewsItem) {

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
	couterName := topic + "-counter"
	coll := client.Database("test").Collection("counters")
	filter := bson.D{{"name", couterName}}

	newData := []interface{}{}
	for i := range data {
		score := getScore(data[i].Title)
		if score == "0" {
			continue
		}
		data[i].Score = score
		newData = append(newData, data[i])
	}
	len := len(newData)
	update := bson.D{{"$inc", bson.D{{"seq", len}}}}

	var prevDoc Counter

	err = coll.FindOneAndUpdate(context.TODO(), filter, update).Decode(&prevDoc)
	if err != nil {
		panic(err)
	}
	seq := prevDoc.Seq

	for i := range newData {
		data[i].ID = seq + 1
	}

	println("Updated seq", prevDoc.Seq)

	coll = client.Database("test").Collection(topic)
	result, err := coll.InsertMany(context.TODO(), newData)
	println("Updated result", result)
	println(result)
	if err != nil {
		panic(err)
	}

}

func connectDB() {
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
}

func getScore(news string) string {
	url := "https://api.baichuan-ai.com/v1/chat/completions"
	apiKey := viper.GetString("BAICHUAN_KEY")

	const noResult = "0"

	data := RequestData{
		Model: "Baichuan4",
		Messages: []Message{
			{
				Role:    "user",
				Content: "请你分析这个关于特斯拉的标题，判断其正面性和负面性，用-5到5的整数表示，-5表示极端负面，5表示极端正面，记住你只返回数字即可：" + news,
			},
		},
		Stream: false, // 这里设置为 false，因为这是一个同步接口
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling data:", err)
		return noResult
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return noResult
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	var fullContent string
	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return noResult
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("请求成功！")
		fmt.Println("请求成功，X-BC-Request-Id:", resp.Header.Get("X-BC-Request-Id"))

		var response Response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Println("Error decoding response:", err)
			return noResult
		}

		// 获取完整的 content

		for _, choice := range response.Choices {
			fullContent += choice.Message.Content
		}

		fmt.Println("完整的内容:", fullContent)
	} else {
		fmt.Println("请求失败，状态码:", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("请求失败，body:", string(body))
		fmt.Println("请求失败，X-BC-Request-Id:", resp.Header.Get("X-BC-Request-Id"))
	}
	return fullContent
}

func testAIProcess() {

	baichuanKey := viper.GetString("BAICHUAN_KEY")
	println(baichuanKey)
	config := openai.DefaultConfig(baichuanKey)
	config.BaseURL = "https://api.baichuan-ai.com/v1/"
	client := openai.NewClientWithConfig(config)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "Baichuan2-Turbo",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Hello!",
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	fmt.Println(resp.Choices[0].Message.Content)
}

func task() {

	topic := viper.GetString("topic")
	duration := viper.GetInt("NEWS_DURATION_HOUR")
	newsItems := collectNews(topic, duration)
	insertData(topic, newsItems)
}

func engine() {

	newsDuration := viper.GetInt("NEWS_DURATION_HOUR")
	ticker := time.NewTicker(time.Duration(newsDuration) * time.Hour)
	defer ticker.Stop()
	for t := range ticker.C {
		fmt.Println("Ticker触发时间:", t)
		task() // 调用要执行的函数
	}

}

func main() {

	viper.SetConfigName("local")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	out := collectNews("tesla", 1)
	println(out)
	// for _, e := range out {
	// 	fmt.Printf("%+v\n", e)
	// }
	// topic := "tesla"

	// println("CreateCounter")
	// createDB(topic)
	// println("CreateDb")
	// insertData(topic, out)
	// println("Inserted")
	// testAIProcess()

}
