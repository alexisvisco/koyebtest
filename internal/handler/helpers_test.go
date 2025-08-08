package handler

import "testing"

func Test_isValidURL(t *testing.T) {
	type args struct {
		testURL string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// Valid URLs
		{
			name: "valid HTTP URL",
			args: args{testURL: "http://example.com"},
			want: true,
		},
		{
			name: "valid HTTPS URL",
			args: args{testURL: "https://example.com"},
			want: true,
		},
		{
			name: "valid URL with path",
			args: args{testURL: "https://example.com/path/to/resource"},
			want: true,
		},
		{
			name: "valid URL with query parameters",
			args: args{testURL: "https://example.com/search?q=test&page=1"},
			want: true,
		},
		{
			name: "valid URL with port",
			args: args{testURL: "https://example.com:8080"},
			want: true,
		},
		{
			name: "valid URL with subdomain",
			args: args{testURL: "https://api.example.com"},
			want: true,
		},
		{
			name: "valid URL with fragment",
			args: args{testURL: "https://example.com/page#section1"},
			want: true,
		},
		{
			name: "valid public IP address",
			args: args{testURL: "http://8.8.8.8"},
			want: true,
		},

		// Invalid URLs - localhost/127.0.0.1
		{
			name: "localhost HTTP",
			args: args{testURL: "http://localhost"},
			want: false,
		},
		{
			name: "localhost HTTPS",
			args: args{testURL: "https://localhost"},
			want: false,
		},
		{
			name: "localhost with port",
			args: args{testURL: "http://localhost:8080"},
			want: false,
		},
		{
			name: "localhost with path",
			args: args{testURL: "http://localhost/api/v1"},
			want: false,
		},
		{
			name: "127.0.0.1 HTTP",
			args: args{testURL: "http://127.0.0.1"},
			want: false,
		},
		{
			name: "127.0.0.1 HTTPS",
			args: args{testURL: "https://127.0.0.1"},
			want: false,
		},
		{
			name: "127.0.0.1 with port",
			args: args{testURL: "http://127.0.0.1:3000"},
			want: false,
		},
		{
			name: "localhost FQDN",
			args: args{testURL: "https://localhost."},
			want: false,
		},

		// Invalid URLs - private IP ranges
		{
			name: "private IP 10.x.x.x",
			args: args{testURL: "http://10.0.0.1"},
			want: false,
		},
		{
			name: "private IP 172.16.x.x",
			args: args{testURL: "https://172.16.0.1"},
			want: false,
		},
		{
			name: "private IP 192.168.x.x",
			args: args{testURL: "http://192.168.1.1"},
			want: false,
		},
		{
			name: "link-local IP",
			args: args{testURL: "http://169.254.1.1"},
			want: false,
		},
		{
			name: "multicast IP",
			args: args{testURL: "http://224.0.0.1"},
			want: false,
		},
		{
			name: "IPv6 loopback",
			args: args{testURL: "http://[::1]"},
			want: false,
		},

		// Invalid URLs - forbidden schemes
		{
			name: "FTP scheme",
			args: args{testURL: "ftp://files.example.com"},
			want: false,
		},
		{
			name: "file scheme",
			args: args{testURL: "file:///etc/passwd"},
			want: false,
		},
		{
			name: "javascript scheme",
			args: args{testURL: "javascript:alert('xss')"},
			want: false,
		},
		{
			name: "data scheme",
			args: args{testURL: "data:text/html,<script>alert('xss')</script>"},
			want: false,
		},

		// Invalid URLs - local domains
		{
			name: "local domain",
			args: args{testURL: "http://myserver.local"},
			want: false,
		},
		{
			name: "localhost domain",
			args: args{testURL: "http://api.localhost"},
			want: false,
		},
		{
			name: "internal domain",
			args: args{testURL: "https://service.internal"},
			want: false,
		},

		// Invalid URLs - malformed
		{
			name: "empty string",
			args: args{testURL: ""},
			want: false,
		},
		{
			name: "invalid URL - no scheme",
			args: args{testURL: "example.com"},
			want: false,
		},
		{
			name: "invalid URL - malformed",
			args: args{testURL: "ht!tp://example.com"},
			want: false,
		},
		{
			name: "invalid URL - spaces",
			args: args{testURL: "http://exam ple.com"},
			want: false,
		},
		{
			name: "invalid URL - just scheme",
			args: args{testURL: "http://"},
			want: false,
		},
		{
			name: "invalid URL - missing host",
			args: args{testURL: "https:///path"},
			want: false,
		},

		// Edge cases
		{
			name: "URL with user info",
			args: args{testURL: "https://user:pass@example.com"},
			want: true,
		},
		{
			name: "URL with international domain",
			args: args{testURL: "https://例え.テスト"},
			want: true,
		},
		{
			name: "very long valid URL",
			args: args{testURL: "https://very-long-domain-name-that-is-still-valid.example.com/very/long/path/with/many/segments"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidURL(tt.args.testURL); got != tt.want {
				t.Errorf("isValidURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
