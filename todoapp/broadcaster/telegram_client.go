package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramClient struct {
	token  string
	chatID string
	client *http.Client
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type TelegramResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

func NewTelegramClient(token, chatID string) *TelegramClient {
	return &TelegramClient{
		token:  token,
		chatID: chatID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *TelegramClient) SendMessage(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)

	payload := TelegramMessage{
		ChatID:    t.chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram request: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode telegram response: %w", err)
	}

	if !telegramResp.Ok {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return nil
}
