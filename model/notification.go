package model

// NotificationRequest is the payload for a text notification.
type NotificationRequest struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

// NotificationResponse identifies a message accepted by WhatsApp.
type NotificationResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}
