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
	Model          string            `json:"model"`
	Messages       []Message         `json:"messages"`
	Stream         bool              `json:"stream"`
	ResponseFormat map[string]string `json:"response_format"`
}

type Choice struct {
	Index          int               `json:"index"`
	Message        Message           `json:"message"`
	ResponseFormat map[string]string `json:"response_format"`
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

const systemPromt = `
The user will provide some news text. Please analyse the title and content, and output in JSON format. Please analyze the content and determine its positivity and negativity about Tesla, using an integer scale from -5 to 5, where -5 indicates extreme negativity, 5 indicates extreme positivity, and 0 indicates neutrality or no impact. If the content is not about the title, please ignore the content just analyse the title. Then use not more than 100 words shows your analyse.

EXAMPLE INPUT: 
Title: Tesla is being sued by the family of a motorcyclist who died in an accident with a car driven by an autopilot
Content: Some news content.

EXAMPLE JSON OUTPUT:
{
    "score": "4",
    "analyse": "This news shows xxxx so XXXX"
}
`

func GetScore(news string) string {
	url := "https://api.baichuan-ai.com/v1/chat/completions"
	apiKey := viper.GetString("BAICHUAN_KEY")

	const noResult = "0"

	data := RequestData{
		Model: "Baichuan4",
		Messages: []Message{
			{
				Role:    "user",
				Content: "请你分析这个关于特斯拉的标题，判断其正面性和负面性，用-5到5的整数表示，-5表示极端负面，5表示极端正面，0表示无关或者没有影响，记住你只返回数字即可：" + news,
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

// Add this new type for DeepSeek's response
type DeepSeekResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func GetScoreDeepSeek(news string) string {
	url := "https://api.deepseek.com/chat/completions"
	apiKey := viper.GetString("DEEPSEEK_KEY")

	const noResult = "0"

	data := RequestData{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPromt,
			},
			{
				Role:    "user",
				Content: news,
			},
		},
		Stream: false,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
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

		var response DeepSeekResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			fmt.Println("Error decoding response:", err)
			return noResult
		}

		if len(response.Choices) > 0 {
			content := response.Choices[0].Message.Content
			fmt.Println("完整的内容:", content)
			return content
		}
		return noResult
	} else {
		fmt.Println("请求失败，状态码:", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("请求失败，body:", string(body))
		return noResult
	}
}
