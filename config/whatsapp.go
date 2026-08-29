package config

import "fmt"

// WhatsAppConfig contains the WhatsApp session storage configuration.
type WhatsAppConfig struct {
	SessionDB string
}

func (c WhatsAppConfig) validate() error {
	if c.SessionDB == "" {
		return fmt.Errorf("whatsapp session database path is required")
	}
	return nil
}
