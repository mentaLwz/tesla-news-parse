package main

import (
	"log"
	"time"

	"github.com/mentaLwz/tesla-news-parse/config"
	"github.com/mentaLwz/tesla-news-parse/lru"
	"github.com/mentaLwz/tesla-news-parse/news"

	"github.com/spf13/viper"
)

func task() {
	topic := viper.GetString("topic")
	duration := viper.GetInt("NEWS_DURATION_HOUR")
	newsItems := news.CollectNews(topic, duration)
	news.InsertData(topic, newsItems)
}

func engine() {
	newsDuration := viper.GetInt("NEWS_DURATION_HOUR")
	ticker := time.NewTicker(time.Duration(newsDuration) * time.Hour)
	defer ticker.Stop()
	for t := range ticker.C {
		log.Println("Ticker触发时间:", t)
		task()
	}
}

func main() {
	config.LoadConfig()
	lru.GetInstance(1000)
	// 启动任务引擎
	engine()
}
