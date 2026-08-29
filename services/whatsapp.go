package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/proto"
)

// ErrWhatsAppDisconnected indicates that the WhatsApp session cannot send messages.
var ErrWhatsAppDisconnected = errors.New("whatsapp client is disconnected")

// ErrInvalidRecipient indicates that a recipient is not an E.164-like phone number.
var ErrInvalidRecipient = errors.New("invalid WhatsApp recipient")

// WhatsAppClient sends text messages through one persisted WhatsApp Web session.
type WhatsAppClient struct {
	client          *whatsmeow.Client
	container       *sqlstore.Container
	log             *slog.Logger
	connectedGauge  metric.Int64ObservableGauge
	loggedInGauge   metric.Int64ObservableGauge
	messagesSent    metric.Int64Counter
	sendDuration    metric.Float64Histogram
	connectionState *connectionState
}

type connectionState struct {
	connected bool
	loggedIn  bool
}

// NewWhatsAppClient creates a client backed by a SQLite session database.
func NewWhatsAppClient(ctx context.Context, sessionDB string, log *slog.Logger, meterProvider metric.MeterProvider) (*WhatsAppClient, error) {
	address := fmt.Sprintf("file:%s?_foreign_keys=on", sessionDB)
	container, err := sqlstore.New(ctx, "sqlite3", address, nil)
	if err != nil {
		return nil, fmt.Errorf("open WhatsApp session database: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("load WhatsApp device: %w", err)
	}

	state := &connectionState{}
	meter := meterProvider.Meter("whatsapp")

	connectedGauge, err := meter.Int64ObservableGauge(
		"whatsapp_connected",
		metric.WithDescription("Whether the WhatsApp client is connected to the server"),
	)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp_connected gauge: %w", err)
	}

	loggedInGauge, err := meter.Int64ObservableGauge(
		"whatsapp_logged_in",
		metric.WithDescription("Whether the WhatsApp client has an active login session"),
	)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp_logged_in gauge: %w", err)
	}

	messagesSent, err := meter.Int64Counter(
		"whatsapp_messages_sent_total",
		metric.WithDescription("Total number of WhatsApp messages sent"),
	)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp_messages_sent_total counter: %w", err)
	}

	sendDuration, err := meter.Float64Histogram(
		"whatsapp_send_duration_seconds",
		metric.WithDescription("Duration of WhatsApp send operations in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("create whatsapp_send_duration_seconds histogram: %w", err)
	}

	c := &WhatsAppClient{
		client:          whatsmeow.NewClient(device, nil),
		container:       container,
		log:             log,
		connectedGauge:  connectedGauge,
		loggedInGauge:   loggedInGauge,
		messagesSent:    messagesSent,
		sendDuration:    sendDuration,
		connectionState: state,
	}

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		value := int64(0)
		if state.connected {
			value = 1
		}
		o.ObserveInt64(connectedGauge, value)

		value = 0
		if state.loggedIn {
			value = 1
		}
		o.ObserveInt64(loggedInGauge, value)
		return nil
	}, connectedGauge, loggedInGauge)
	if err != nil {
		return nil, fmt.Errorf("register whatsapp connection callback: %w", err)
	}

	return c, nil
}

// Connect connects an existing session or prints a QR code for first-time pairing.
func (c *WhatsAppClient) Connect(ctx context.Context) error {
	if c.client.Store.ID == nil {
		qrEvents, err := c.client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("create WhatsApp QR channel: %w", err)
		}
		go c.printQREvents(ctx, qrEvents)
	}

	if err := c.client.ConnectContext(ctx); err != nil {
		return fmt.Errorf("connect WhatsApp client: %w", err)
	}

	c.connectionState.connected = c.client.IsConnected()
	c.connectionState.loggedIn = c.client.IsLoggedIn()
	return nil
}

func (c *WhatsAppClient) printQREvents(ctx context.Context, events <-chan whatsmeow.QRChannelItem) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Event == "code" {
				c.log.Info("Scan the QR code to link WhatsApp")
				qrterminal.GenerateHalfBlock(event.Code, qrterminal.L, os.Stdout)
				continue
			}
			c.log.Info("WhatsApp pairing event", "event", event.Event)
		}
	}
}

// Send sends a text notification and returns the WhatsApp message ID.
func (c *WhatsAppClient) Send(ctx context.Context, recipient, message string) (string, error) {
	start := time.Now()

	phone, err := normalizePhone(recipient)
	if err != nil {
		c.recordSend(ctx, "invalid_recipient", time.Since(start).Seconds())
		return "", err
	}
	if !c.client.IsConnected() || !c.client.IsLoggedIn() {
		c.recordSend(ctx, "disconnected", time.Since(start).Seconds())
		return "", ErrWhatsAppDisconnected
	}

	response, err := c.client.SendMessage(ctx, types.NewJID(phone, types.DefaultUserServer), &waE2E.Message{
		Conversation: proto.String(message),
	})
	if err != nil {
		c.recordSend(ctx, "provider_error", time.Since(start).Seconds())
		return "", fmt.Errorf("send WhatsApp message: %w", err)
	}

	c.recordSend(ctx, "success", time.Since(start).Seconds())
	return string(response.ID), nil
}

func (c *WhatsAppClient) recordSend(ctx context.Context, result string, durationSeconds float64) {
	c.messagesSent.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	c.sendDuration.Record(ctx, durationSeconds, metric.WithAttributes(attribute.String("result", result)))
}

func normalizePhone(recipient string) (string, error) {
	phone := strings.TrimPrefix(strings.TrimSpace(recipient), "+")
	if len(phone) < 8 || len(phone) > 15 {
		return "", fmt.Errorf("%w: must contain 8 to 15 digits including country code", ErrInvalidRecipient)
	}
	for _, char := range phone {
		if !unicode.IsDigit(char) || char > unicode.MaxASCII {
			return "", fmt.Errorf("%w: must contain only digits with an optional leading +", ErrInvalidRecipient)
		}
	}
	return phone, nil
}

// Close disconnects WhatsApp and closes its session database.
func (c *WhatsAppClient) Close() error {
	c.client.Disconnect()
	c.connectionState.connected = false
	c.connectionState.loggedIn = false
	if err := c.container.Close(); err != nil {
		return fmt.Errorf("close WhatsApp session database: %w", err)
	}
	return nil
}
