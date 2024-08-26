//go:build alt

// go run -tags alt main_alt.go

package main

import (
	"github.com/mentaLwz/tesla-news-parse/config"
	"github.com/mentaLwz/tesla-news-parse/news"
	"github.com/spf13/viper"
)

func main() {
	config.LoadConfig()
	// 设置配置
	topic := viper.GetString("topic")
	// duration := viper.GetInt("NEWS_DURATION_HOUR")
	news.UpdateAnalyseFieldInDatabase(topic)
}
