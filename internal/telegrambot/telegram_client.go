package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTelegramAPIBaseURL = "https://api.telegram.org"

type TelegramClient struct {
	baseURL    string
	botToken   string
	httpClient *http.Client
}

type Update struct {
	ID      int64    `json:"update_id"`
	Message *Message `json:"message"`
}

type Message struct {
	ID   int64  `json:"message_id"`
	Text string `json:"text"`
	Chat Chat   `json:"chat"`
}

type Chat struct {
	ID int64 `json:"id"`
}

func NewTelegramClient(botToken string, httpClient *http.Client) *TelegramClient {
	return NewTelegramClientWithBaseURL(defaultTelegramAPIBaseURL, botToken, httpClient)
}

func NewTelegramClientWithBaseURL(baseURL string, botToken string, httpClient *http.Client) *TelegramClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &TelegramClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		botToken:   botToken,
		httpClient: httpClient,
	}
}

func (c *TelegramClient) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	query := url.Values{}
	query.Set("offset", fmt.Sprintf("%d", offset))
	query.Set("timeout", fmt.Sprintf("%.0f", timeout.Seconds()))

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "getUpdates?"+query.Encode(), nil, &result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram getUpdates returned ok=false")
	}

	return result.Result, nil
}

func (c *TelegramClient) SendMessage(ctx context.Context, chatID int64, text string) error {
	var result struct {
		OK bool `json:"ok"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "sendMessage", map[string]any{
		"chat_id": chatID,
		"text":    text,
	}, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram sendMessage returned ok=false")
	}

	return nil
}

func (c *TelegramClient) doJSON(ctx context.Context, method string, methodPath string, body any, result any) error {
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode telegram request body: %w", err)
		}
		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.methodURL(methodPath), requestBody)
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram returned status %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode telegram response body: %w", err)
	}

	return nil
}

func (c *TelegramClient) methodURL(methodPath string) string {
	return fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.botToken, methodPath)
}
