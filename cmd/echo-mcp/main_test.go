package main

import "testing"

func TestHTTPAddrFromEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "default address when unset",
			env:  map[string]string{},
			want: ":8080",
		},
		{
			name: "configured address when set",
			env: map[string]string{
				"ECHO_MCP_HTTP_ADDR": "127.0.0.1:18080",
			},
			want: "127.0.0.1:18080",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := httpAddrFromEnvironment(func(key string) string {
				return tt.env[key]
			})

			if got != tt.want {
				t.Fatalf("httpAddrFromEnvironment() = %q, want %q", got, tt.want)
			}
		})
	}
}
