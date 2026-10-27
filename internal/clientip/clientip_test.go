package clientip

import (
	"net/http/httptest"
	"testing"
)

func TestFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name: "CF-Connecting-IP wins over spoofed X-Forwarded-For",
			headers: map[string]string{
				"CF-Connecting-IP": "198.51.100.7",
				"X-Forwarded-For":  "1.2.3.4, 198.51.100.7",
			},
			want: "198.51.100.7",
		},
		{
			name:    "falls back to first X-Forwarded-For entry",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9, 10.0.0.1"},
			want:    "203.0.113.9",
		},
		{
			name:    "falls back to X-Real-IP",
			headers: map[string]string{"X-Real-IP": "203.0.113.10"},
			want:    "203.0.113.10",
		},
		{
			name: "falls back to RemoteAddr without port",
			want: "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			if got := FromRequest(r); got != tt.want {
				t.Errorf("FromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}
