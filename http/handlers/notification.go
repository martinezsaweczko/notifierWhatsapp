package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"whatsapp-notifier/http/middleware"
	"whatsapp-notifier/model"
	"whatsapp-notifier/services"
)

// NotificationSender sends a text notification to a recipient.
type NotificationSender interface {
	Send(ctx context.Context, recipient, message string) (string, error)
}

// NotificationHandlerConfig contains dependencies for NotificationHandler.
type NotificationHandlerConfig struct {
	Log      *slog.Logger
	Sender   NotificationSender
	BasePath string
}

// NotificationHandler handles WhatsApp notification requests.
type NotificationHandler struct {
	log      *slog.Logger
	sender   NotificationSender
	basePath string
}

// NewNotificationHandler creates a notification handler.
func (c NotificationHandlerConfig) NewNotificationHandler() (*NotificationHandler, error) {
	if c.Log == nil {
		return nil, fmt.Errorf("notification handler logger is required")
	}
	if c.Sender == nil {
		return nil, fmt.Errorf("notification sender is required")
	}
	return &NotificationHandler{log: c.Log, sender: c.Sender, basePath: c.BasePath}, nil
}

// RegisterRoutes registers the notification endpoint.
func (h *NotificationHandler) RegisterRoutes(router Router, middlewares ...middleware.Middleware) {
	path := fmt.Sprintf("%s/notifications", h.basePath)
	router.HandleFuncWithMiddleware("POST "+path, h.send, middlewares...)
}

// Send a WhatsApp text notification.
//
//	@Summary      Send a WhatsApp notification
//	@Tags         notifications
//	@Accept       json
//	@Produce      json
//	@Param        request body model.NotificationRequest true "Notification"
//	@Success      202 {object} model.NotificationResponse
//	@Failure      400 {object} model.ErrorResponse
//	@Failure      502 {object} model.ErrorResponse
//	@Failure      503 {object} model.ErrorResponse
//	@Router       /notifications [post]
func (h *NotificationHandler) send(w http.ResponseWriter, r *http.Request) {
	var request model.NotificationRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid request body"})
		return
	}

	request.Recipient = strings.TrimSpace(request.Recipient)
	request.Message = strings.TrimSpace(request.Message)
	if request.Recipient == "" || request.Message == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "recipient and message are required"})
		return
	}

	messageID, err := h.sender.Send(r.Context(), request.Recipient, request.Message)
	if errors.Is(err, services.ErrWhatsAppDisconnected) {
		writeJSON(w, http.StatusServiceUnavailable, model.ErrorResponse{Error: "WhatsApp is not connected"})
		return
	}
	if errors.Is(err, services.ErrInvalidRecipient) {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "recipient must contain 8 to 15 digits including country code"})
		return
	}
	if err != nil {
		h.log.Error("Failed to send WhatsApp notification", "error", err)
		writeJSON(w, http.StatusBadGateway, model.ErrorResponse{Error: "failed to send notification"})
		return
	}

	h.log.Info("WhatsApp notification accepted", "message_id", messageID)
	writeJSON(w, http.StatusAccepted, model.NotificationResponse{MessageID: messageID, Status: "sent"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
