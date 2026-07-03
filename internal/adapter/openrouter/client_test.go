package openrouter

import "testing"

func TestParsePick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantID  int64
		wantErr bool
	}{
		{
			name:    "clean json",
			content: `{"appId": 440, "why": "Hats."}`,
			wantID:  440,
		},
		{
			name:    "markdown fenced",
			content: "```json\n{\"appId\": 570, \"why\": \"MOBA night.\"}\n```",
			wantID:  570,
		},
		{
			name:    "chatter around json",
			content: `Sure! Here's my pick: {"appId": 730, "why": "Clutch or kick."} Enjoy!`,
			wantID:  730,
		},
		{
			name:    "no json",
			content: "I recommend Half-Life 2, a classic.",
			wantErr: true,
		},
		{
			name:    "missing why",
			content: `{"appId": 10}`,
			wantErr: true,
		},
		{
			name:    "zero appid",
			content: `{"appId": 0, "why": "nothing"}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, why, err := parsePick(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got id=%d why=%q", id, why)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if id != tt.wantID {
				t.Fatalf("id = %d, want %d", id, tt.wantID)
			}
			if why == "" {
				t.Fatal("why empty")
			}
		})
	}
}
