package main

import (
	"bytes"         // для работы с байтовыми буферами(нужно при отправке запроса)
	"encoding/json" // преобразует данные в JSON и обратно
	"fmt"           // форматирование строк
	"io/ioutil"     // читает ответы от сервера
	"log"
	"net/http" // отправлять http- запросы
	"os"       // для работы с переменными окружения (.env)

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type Message struct {
	Role string `json:"role"` // кто говорит : "user", "system", "assistant"
	Text string `json:"text"` // текст сообщения
}

type CompletionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type GPTRequest struct {
	ModelUri          string            `json:"modelUri"`
	CompletionOptions CompletionOptions `json:"completionOptions"`
	Messages          []Message         `json:"messages"`
}

func askYandexGPT(userText string) (string, error) {
	apiKey := os.Getenv("YANDEX_API_KEY")
	folderID := os.Getenv("YANDEX_FOLDER_ID")

	url := "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"

	requestBody := GPTRequest{
		ModelUri: fmt.Sprintf("gpt://%s/yandexgpt-lite", folderID),
		CompletionOptions: CompletionOptions{
			Stream:      false,
			Temperature: 0.7,
			MaxTokens:   100,
		},
		Messages: []Message{
			{Role: "user", Text: userText},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Api-Key "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-folder-id", folderID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	// Парсим ответ
	text := "🤖 Ответ не получен"
	if r, ok := result["result"].(map[string]interface{}); ok {
		if alternatives, ok := r["alternatives"].([]interface{}); ok && len(alternatives) > 0 {
			msg := alternatives[0].(map[string]interface{})["message"].(map[string]interface{})
			text = msg["text"].(string)
		}
	}

	return text, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("✅ Бот запущен: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userMessage := update.Message.Text
		replyText, err := askYandexGPT(userMessage)
		if err != nil {
			replyText = "⚠️ Ошибка: " + err.Error()
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		bot.Send(msg)
	}
}
