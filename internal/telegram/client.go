package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	token      string
	miniAppURL string
	httpClient *http.Client
}

type Feedback struct {
	UserID    int64
	FirstName string
	Username  string
	Category  string
	Message   string
	Screen    string
	TopicSlug string
}

func NewClient(token, miniAppURL string) *Client {
	return &Client{
		token:      token,
		miniAppURL: miniAppURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c.token != "" && c.miniAppURL != ""
}

func (c *Client) SendStart(ctx context.Context, chatID int64) error {
	return c.sendMessage(ctx, map[string]any{
		"chat_id": chatID,
		"text":    "Добро пожаловать в Lison. Изучайте русский язык по темам и повторяйте слова вовремя.",
		"reply_markup": map[string]any{
			"inline_keyboard": [][]any{{map[string]any{
				"text":    "Открыть Lison",
				"web_app": map[string]string{"url": c.miniAppURL},
			}}},
		},
	})
}

func (c *Client) SendReminder(ctx context.Context, chatID int64, cardCount, grammarCount int) error {
	text := "Пора повторить материал."
	screen := "review"
	switch {
	case cardCount > 0 && grammarCount > 0:
		text = fmt.Sprintf("Пора повторить: %d %s и %d %s по грамматике.", cardCount, russianCardNoun(cardCount), grammarCount, russianTopicNoun(grammarCount))
	case grammarCount > 0:
		text = fmt.Sprintf("Пора освежить грамматику. Вас ждут %d %s — всего несколько минут игры.", grammarCount, russianTopicNoun(grammarCount))
		screen = "grammar"
	default:
		text = fmt.Sprintf("Пора повторить слова. Сегодня вас ждут %d %s.", cardCount, russianCardNoun(cardCount))
	}
	return c.sendMessage(ctx, map[string]any{
		"chat_id": chatID,
		"text":    text,
		"reply_markup": map[string]any{
			"inline_keyboard": [][]any{{map[string]any{
				"text":    "Начать повторение",
				"web_app": map[string]string{"url": c.miniAppURL + "?screen=" + screen},
			}}},
		},
	})
}

func russianTopicNoun(count int) string {
	mod100 := count % 100
	mod10 := mod100 % 10
	if mod100 > 10 && mod100 < 20 {
		return "тем"
	}
	if mod10 == 1 {
		return "тема"
	}
	if mod10 >= 2 && mod10 <= 4 {
		return "темы"
	}
	return "тем"
}

func (c *Client) SendFeedback(ctx context.Context, chatID int64, feedback Feedback) error {
	username := "—"
	if feedback.Username != "" {
		username = "@" + feedback.Username
	}
	contextLine := feedback.Screen
	if feedback.TopicSlug != "" {
		contextLine += " / " + feedback.TopicSlug
	}
	if contextLine == "" {
		contextLine = "—"
	}

	return c.sendMessage(ctx, map[string]any{
		"chat_id": chatID,
		"text": fmt.Sprintf(
			"💬 Новый отзыв Lison\n\nКатегория: %s\nЭкран: %s\nПользователь: %s (%s, %d)\n\n%s",
			feedback.Category, contextLine, feedback.FirstName, username, feedback.UserID, feedback.Message,
		),
	})
}

func russianCardNoun(count int) string {
	mod100 := count % 100
	mod10 := mod100 % 10
	if mod100 > 10 && mod100 < 20 {
		return "карточек"
	}
	if mod10 == 1 {
		return "карточка"
	}
	if mod10 >= 2 && mod10 <= 4 {
		return "карточки"
	}
	return "карточек"
}

func (c *Client) sendMessage(ctx context.Context, payload map[string]any) error {
	if !c.Enabled() {
		return fmt.Errorf("telegram client is not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+c.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage returned %s", resp.Status)
	}
	return nil
}
