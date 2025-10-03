package main

import "testing"

func TestDeriveGatewayDialTarget(t *testing.T) {
	tests := map[string]struct {
		listenAddr string
		want       string
		wantErr    bool
	}{
		"default listen":      {listenAddr: ":9090", want: "127.0.0.1:9090"},
		"all interfaces":      {listenAddr: "0.0.0.0:50051", want: "127.0.0.1:50051"},
		"localhost preserved": {listenAddr: "localhost:6000", want: "localhost:6000"},
		"ipv6 any":            {listenAddr: "[::]:7000", want: "127.0.0.1:7000"},
		"missing port":        {listenAddr: "localhost", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := deriveGatewayDialTarget(tc.listenAddr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("deriveGatewayDialTarget returned unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("deriveGatewayDialTarget(%q) = %q, want %q", tc.listenAddr, got, tc.want)
			}
		})
	}
}
