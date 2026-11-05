package alerting

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelegramPreviewUsesSubmittedSettingsWithoutSaving(t *testing.T) {
	s, id := setup(t)
	_, err := s.UpdateTelegram(0, TelegramUpdate{Token: ptr("123:global-token")})
	require.NoError(t, err)
	_, err = s.UpdateTelegram(id, TelegramUpdate{Enabled: true, ChatID: "-100123", Token: ptr("456:project-token")})
	require.NoError(t, err)
	before, err := s.Store.GetAlertChannel(id, Telegram)
	require.NoError(t, err)
	globalBefore, err := s.Store.GetAlertChannel(0, Telegram)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, wantToken, locale string
		scope                   int
		token                   *string
	}{
		{"saved project token", "456:project-token", "en", id, nil},
		{"inherit global", "123:global-token", "ru", id, ptr("")},
		{"unsaved override", "789:new-token", "ru-RU", id, ptr(" 789:new-token ")},
		{"saved global token", "123:global-token", "en", 0, nil},
		{"unsaved global token", "789:new-token", "en", 0, ptr("789:new-token")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "/bot"+tc.wantToken+"/sendMessage", r.URL.Path)
				require.Equal(t, "POST", r.Method)
				var body map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "@preview", body["chat_id"])
				if strings.HasPrefix(tc.locale, "ru") {
					require.Contains(t, body["text"], "тестовое оповещение")
				} else {
					require.Contains(t, body["text"], "test notification")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
			})}
			require.NoError(t, s.TestTelegram(context.Background(), tc.scope, TelegramUpdate{ChatID: " @preview ", Token: tc.token}, tc.locale, client))
			require.Equal(t, 1, calls)
			after, err := s.Store.GetAlertChannel(id, Telegram)
			require.NoError(t, err)
			require.Equal(t, before, after)
			globalAfter, err := s.Store.GetAlertChannel(0, Telegram)
			require.NoError(t, err)
			require.Equal(t, globalBefore, globalAfter)
		})
	}
}

func TestTelegramPreviewValidationAndSafeFailures(t *testing.T) {
	s, id := setup(t)
	calls := 0
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New(r.URL.String())
	})}
	for _, update := range []TelegramUpdate{
		{ChatID: "", Token: ptr("123:private-token")},
		{ChatID: "@preview", Token: ptr("123:private/token")},
		{ChatID: "@preview"},
	} {
		err := s.TestTelegram(context.Background(), id, update, "en", client)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private")
	}
	require.Zero(t, calls)
	err := s.TestTelegram(context.Background(), id, TelegramUpdate{ChatID: "@preview", Token: ptr("123:private-token")}, "en", client)
	require.EqualError(t, err, "Telegram request failed (network error or timeout)")
	require.Equal(t, 1, calls)
	ctx, cancel := context.WithCancel(context.Background())
	client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		require.ErrorIs(t, r.Context().Err(), context.Canceled)
		return nil, r.Context().Err()
	})
	cancel()
	require.Error(t, s.TestTelegram(ctx, id, TelegramUpdate{ChatID: "@preview", Token: ptr("123:private-token")}, "en", client))
}

func TestWorkflowNotificationUsesProjectChannelPlainTextAndLink(t *testing.T) {
	s, id := setup(t)
	calls := 0
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/bot123:project-token/sendMessage", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "@workflow", body["chat_id"])
		require.Equal(t, "<b>plain & text</b>", body["text"])
		require.NotContains(t, body, "parse_mode")
		keyboard := body["reply_markup"].(map[string]any)["inline_keyboard"].([]any)
		button := keyboard[0].([]any)[0].(map[string]any)
		require.Equal(t, "https://texec.example.test/project/1/workflows/2/runs/3", button["url"])
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	send := func() error {
		return s.SendWorkflowNotification(context.Background(), id, "<b>plain & text</b>", "https://texec.example.test/project/1/workflows/2/runs/3", "Open run", client)
	}
	require.EqualError(t, send(), "Telegram notifications are disabled for this project")
	require.Zero(t, calls)
	_, err := s.UpdateTelegram(id, TelegramUpdate{Enabled: true, ChatID: "@workflow", Token: ptr("123:project-token")})
	require.NoError(t, err)
	require.NoError(t, send())
	require.Equal(t, 1, calls)
}

func TestTelegramTopicsRouteAllNotificationKinds(t *testing.T) {
	s, id := setup(t)
	_, err := s.UpdateTelegram(0, TelegramUpdate{Token: ptr("123:global-token")})
	require.NoError(t, err)
	_, err = s.UpdateTelegram(id, TelegramUpdate{Enabled: true, ChatID: "-100123", MessageThreadID: new(int64(42))})
	require.NoError(t, err)
	before, err := s.Store.GetAlertChannel(id, Telegram)
	require.NoError(t, err)
	for _, tc := range []struct {
		name  string
		topic int64
		send  func(*http.Client) error
	}{
		{"legacy alert", 42, func(c *http.Client) error { _, err := s.SendTelegram(id, "legacy", c); return err }},
		{"task result", 42, func(c *http.Client) error {
			_, err := s.SendTaskNotification(context.Background(), id, "result", c)
			return err
		}},
		{"workflow and approval", 42, func(c *http.Client) error {
			return s.SendWorkflowNotification(context.Background(), id, "approval", "https://texec.example.test/run", "Open", c)
		}},
		{"unsaved project test", 73, func(c *http.Client) error {
			return s.TestTelegram(context.Background(), id, TelegramUpdate{ChatID: "-100123", MessageThreadID: new(int64(73))}, "en", c)
		}},
		{"test without topic", 0, func(c *http.Client) error {
			return s.TestTelegram(context.Background(), id, TelegramUpdate{ChatID: "-100123"}, "en", c)
		}},
		{"global credential test", 99, func(c *http.Client) error {
			return s.TestTelegram(context.Background(), 0, TelegramUpdate{ChatID: "-100123", MessageThreadID: new(int64(99))}, "en", c)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "/bot123:global-token/sendMessage", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "-100123", body["chat_id"])
				if tc.topic == 0 {
					require.NotContains(t, body, "message_thread_id")
				} else {
					require.Equal(t, float64(tc.topic), body["message_thread_id"])
				}
				if tc.name == "workflow and approval" {
					require.Contains(t, body, "reply_markup")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
			})}
			require.NoError(t, tc.send(client))
			require.Equal(t, 1, calls)
			after, err := s.Store.GetAlertChannel(id, Telegram)
			require.NoError(t, err)
			require.Equal(t, before, after, "preview must not change the saved topic")
		})
	}
}

func TestTelegramInvalidTopicDoesNotSendOrFallBack(t *testing.T) {
	s, id := setup(t)
	_, err := s.UpdateTelegram(id, TelegramUpdate{Enabled: true, ChatID: "-100123", MessageThreadID: new(int64(42)), Token: ptr("123:private-token")})
	require.NoError(t, err)
	calls := 0
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, float64(42), body["message_thread_id"])
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"ok":false,"description":"Bad Request: message thread not found"}`))}, nil
	})}
	for _, topic := range []int64{-1, 2147483648} {
		err := s.TestTelegram(context.Background(), id, TelegramUpdate{ChatID: "-100123", MessageThreadID: &topic}, "en", client)
		require.ErrorContains(t, err, "topic ID")
	}
	require.Zero(t, calls)
	sent, err := s.SendTaskNotification(context.Background(), id, "result", client)
	require.False(t, sent)
	require.EqualError(t, err, "Telegram request failed (HTTP 400)")
	require.Equal(t, 1, calls, "a missing topic must never fall back to the general chat")
}
