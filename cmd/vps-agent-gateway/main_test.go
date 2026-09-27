package main

import "testing"

func TestListenIsLoopback(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"[::1]:8080", true},
		{"localhost:8080", true},
		{":8080", false},
		{"0.0.0.0:8080", false},
		{"[::]:8080", false},
		{"example.com:8080", false},
		{"not-an-address", false},
	}
	for _, tt := range tests {
		if got := listenIsLoopback(tt.addr); got != tt.want {
			t.Errorf("listenIsLoopback(%q)=%v, want %v", tt.addr, got, tt.want)
		}
	}
}
