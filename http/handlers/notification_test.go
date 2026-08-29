package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whatsapp-notifier/services"
)

type notificationSenderStub struct {
	messageID string
	err       error
}

func (s notificationSenderStub) Send(context.Context, string, string) (string, error) {
	return s.messageID, s.err
}

func TestNotificationHandlerSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		body   string
		sender notificationSenderStub
		status int
	}{
		{name: "accepted", body: `{"recipient":"15551234567","message":"hello"}`, sender: notificationSenderStub{messageID: "message-1"}, status: http.StatusAccepted},
		{name: "invalid JSON", body: `{`, status: http.StatusBadRequest},
		{name: "missing message", body: `{"recipient":"15551234567"}`, status: http.StatusBadRequest},
		{name: "disconnected", body: `{"recipient":"15551234567","message":"hello"}`, sender: notificationSenderStub{err: services.ErrWhatsAppDisconnected}, status: http.StatusServiceUnavailable},
		{name: "invalid recipient", body: `{"recipient":"invalid","message":"hello"}`, sender: notificationSenderStub{err: services.ErrInvalidRecipient}, status: http.StatusBadRequest},
		{name: "provider error", body: `{"recipient":"15551234567","message":"hello"}`, sender: notificationSenderStub{err: errors.New("send failed")}, status: http.StatusBadGateway},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := (NotificationHandlerConfig{
				Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
				Sender: test.sender,
			}).NewNotificationHandler()
			if err != nil {
				t.Fatalf("NewNotificationHandler() error = %v", err)
			}

			request := httptest.NewRequest(http.MethodPost, "/notifications", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.send(response, request)
			if response.Code != test.status {
				t.Errorf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}
