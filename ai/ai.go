package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

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

func GetScore(news string) string {
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
		Stream: false,
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

		var fullContent string
		for _, choice := range response.Choices {
			fullContent += choice.Message.Content
		}

		fmt.Println("完整的内容:", fullContent)
		return fullContent
	} else {
		fmt.Println("请求失败，状态码:", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("请求失败，body:", string(body))
		fmt.Println("请求失败，X-BC-Request-Id:", resp.Header.Get("X-BC-Request-Id"))
		return noResult
	}
}
