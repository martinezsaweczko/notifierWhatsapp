package services

import "testing"

func TestNormalizePhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		recipient string
		want      string
		wantErr   bool
	}{
		{name: "digits", recipient: "15551234567", want: "15551234567"},
		{name: "leading plus", recipient: "+15551234567", want: "15551234567"},
		{name: "spaces", recipient: " +15551234567 ", want: "15551234567"},
		{name: "too short", recipient: "123", wantErr: true},
		{name: "punctuation", recipient: "+1-555-123-4567", wantErr: true},
		{name: "non ASCII digits", recipient: "\u0661\u0665\u0665\u0665\u0661\u0662\u0663\u0664\u0665\u0666\u0667", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizePhone(test.recipient)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizePhone() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("normalizePhone() = %q, want %q", got, test.want)
			}
		})
	}
}
