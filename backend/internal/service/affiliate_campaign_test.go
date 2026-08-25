package service

import "testing"

func TestMaskLeaderboardEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "normal", email: "alice@example.com", want: "al***@example.com"},
		{name: "short local", email: "a@example.com", want: "a***@example.com"},
		{name: "invalid", email: "invalid", want: "***"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := maskLeaderboardEmail(test.email); got != test.want {
				t.Fatalf("maskLeaderboardEmail(%q) = %q, want %q", test.email, got, test.want)
			}
		})
	}
}
