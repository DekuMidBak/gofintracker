package telegrambot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTelegramClientGetUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/bottest-token/getUpdates" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		if r.URL.Query().Get("offset") != "42" || r.URL.Query().Get("timeout") != "30" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}

		writeTestJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"result": []Update{
				{
					ID: 42,
					Message: &Message{
						Text: "/start",
						Chat: Chat{ID: 100},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewTelegramClientWithBaseURL(server.URL, "test-token", server.Client())
	updates, err := client.GetUpdates(context.Background(), 42, 30*time.Second)
	if err != nil {
		t.Fatalf("get updates: %v", err)
	}

	if len(updates) != 1 || updates[0].ID != 42 || updates[0].Message.Chat.ID != 100 {
		t.Fatalf("unexpected updates: %+v", updates)
	}
}

func TestTelegramClientSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/bottest-token/sendMessage" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if body["chat_id"] != float64(100) || body["text"] != "hello" {
			t.Fatalf("unexpected request body: %+v", body)
		}

		writeTestJSON(w, http.StatusOK, map[string]any{
			"ok": true,
		})
	}))
	defer server.Close()

	client := NewTelegramClientWithBaseURL(server.URL, "test-token", server.Client())
	if err := client.SendMessage(context.Background(), 100, "hello"); err != nil {
		t.Fatalf("send message: %v", err)
	}
}
