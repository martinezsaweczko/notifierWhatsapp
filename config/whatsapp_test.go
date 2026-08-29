package config

import "testing"

func TestWhatsAppConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  WhatsAppConfig
		wantErr bool
	}{
		{name: "valid", config: WhatsAppConfig{SessionDB: "whatsapp.db"}},
		{name: "missing database", config: WhatsAppConfig{}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.config.validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
