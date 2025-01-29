package qrng

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetRandomBits(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"uint8","length":1,"data":[255],"success":true}`)
		}))
		defer server.Close()

		client := NewClient()
		client.APIEndpoint = server.URL

		bits, err := client.GetRandomBits(8)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expected := []int{1, 1, 1, 1, 1, 1, 1, 1}
		if fmt.Sprint(bits) != fmt.Sprint(expected) {
			t.Errorf("Expected %v, got %v", expected, bits)
		}
	})

	t.Run("invalid numBits", func(t *testing.T) {
		client := NewClient()
		_, err := client.GetRandomBits(0)
		if err == nil {
			t.Error("Expected error for invalid numBits")
		}
	})
}

func TestGetRandomUint8(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"uint8","length":2,"data":[123,255],"success":true}`)
		}))
		defer server.Close()

		client := NewClient()
		client.APIEndpoint = server.URL

		bytes, err := client.GetRandomUint8(2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expected := []uint8{123, 255}
		if fmt.Sprint(bytes) != fmt.Sprint(expected) {
			t.Errorf("Expected %v, got %v", expected, bytes)
		}
	})
}

func TestGetRandomNumber(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"uint8","length":1,"data":[127],"success":true}`)
		}))
		defer server.Close()

		client := NewClient()
		client.APIEndpoint = server.URL

		num, err := client.GetRandomNumber(0, 255)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if num < 0 || num > 255 {
			t.Errorf("Number %d out of range", num)
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		client := NewClient()
		_, err := client.GetRandomNumber(10, 5)
		if err != ErrInvalidRange {
			t.Errorf("Expected ErrInvalidRange, got %v", err)
		}
	})
}

func TestClientConfiguration(t *testing.T) {
	t.Run("custom HTTP client", func(t *testing.T) {
		client := NewClient()
		client.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	})
}
