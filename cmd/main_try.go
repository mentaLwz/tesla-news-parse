//go:build try
// +build try

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

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

func main() {
	// Example usage
	sourceURL := "https://news.google.com/rss/articles/CBMiqwFBVV95cUxNMTRqdUZpNl9hQldXbGo2YVVLOGFQdkFLYldlMUxUVlNEaElsYjRRODVUMkF3R1RYdWxvT1NoVzdUYS0xSHg3eVdpTjdVODQ5cVJJLWt4dk9vZFBScVp2ZmpzQXZZRy1ncDM5c2tRbXBVVHVrQnpmMGVrQXNkQVItV3h4dVQ1V1BTbjhnM3k2ZUdPdnhVOFk1NmllNTZkdGJTbW9NX0k5U3E2Tkk?oc=5"
	decodedURL, err := decodeGoogleNewsURL(sourceURL)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(decodedURL)
}
