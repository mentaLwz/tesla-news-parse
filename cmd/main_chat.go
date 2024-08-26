//go:build chat
// +build chat

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network" // Updated import
	"github.com/chromedp/chromedp"
	"github.com/go-shiori/go-readability"
	"github.com/mentaLwz/tesla-news-parse/config"
	db "github.com/mentaLwz/tesla-news-parse/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NewsItem struct {
	ID     primitive.ObjectID `bson:"_id"`
	Title  string             `bson:"title"`
	Date   time.Time          `bson:"date"`
	Link   string             `bson:"link"`
	Source string             `bson:"source"`
	Score  string             `bson:"score"`
}

func main() {
	config.LoadConfig()
	client := db.Connect()
	defer db.Disconnect(client)

	// Get the news collection
	newsCollection := client.Database("test").Collection("tesla")

	// Create a context (you might want to use a timeout context in production)
	ctx := context.Background()

	// Query all documents in the news collection
	cursor, err := newsCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Error querying database: %v", err)
	}
	defer cursor.Close(ctx)

	// Iterate through the cursor and process each news item
	for cursor.Next(ctx) {
		var newsItem NewsItem
		if err := cursor.Decode(&newsItem); err != nil {
			log.Printf("Error decoding news item: %v", err)
			continue
		}

		// Process each news item

		processNewsItem(newsItem)
	}

	if err := cursor.Err(); err != nil {
		log.Fatalf("Cursor error: %v", err)
	}
}

func processNewsItem(newsItem NewsItem) {
	fmt.Printf("Processing news item: %s\n", newsItem.Title)

	// Create a new context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set a timeout for the entire operation
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var content string
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1280, 720),
		chromedp.Navigate(newsItem.Link),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		network.SetExtraHTTPHeaders(map[string]interface{}{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		}),
		chromedp.OuterHTML("body", &content, chromedp.ByQuery),
	)

	if err != nil {
		log.Printf("Error fetching content from %s: %v", newsItem.Link, err)
		return
	}

	// Parse the article using go-readability
	article, err := readability.FromReader(strings.NewReader(content), nil)
	if err != nil {
		log.Printf("Error parsing article from %s: %v", newsItem.Link, err)
		return
	}

	// Extract and clean the main content
	cleanedContent := cleanContent(article.TextContent)

	// Truncate content if necessary
	// if len(cleanedContent) > 500 {
	// 	cleanedContent = cleanedContent[:497] + "..."
	// }
	fmt.Println("================================================================================")
	fmt.Printf("ID: %s\nTitle: %s\nDate: %v\nLink: %s\nSource: %s\nScore: %s\nContent: %s\n\n",
		newsItem.ID.Hex(), newsItem.Title, newsItem.Date, newsItem.Link, newsItem.Source, newsItem.Score, cleanedContent)
}

func cleanContent(content string) string {
	// Split content into words
	words := strings.Fields(content)

	// Filter out unwanted words and join
	var cleanWords []string
	for _, word := range words {
		if len(word) > 1 && !strings.HasPrefix(word, "function") && !strings.HasPrefix(word, "var") {
			cleanWords = append(cleanWords, word)
		}
	}

	return strings.Join(cleanWords, " ")
}
