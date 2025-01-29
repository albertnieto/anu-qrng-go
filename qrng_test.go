package qrng_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qrng "github.com/albertnieto/anu-qrng-go"
)

func TestGetRandomBits(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"uint8","length":1,"data":[255],"success":true}`)
		}))
		defer server.Close()

		client := qrng.NewClient()
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
		client := qrng.NewClient()
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

		client := qrng.NewClient()
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

func TestGetRandomUint16(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"uint16","length":2,"data":[32767,65535],"success":true}`)
		}))
		defer server.Close()

		client := qrng.NewClientWithAPIKey("---")
		client.APIEndpoint = server.URL

		shorts, err := client.GetRandomUint16(2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expected := []uint16{32767, 65535}
		if fmt.Sprint(shorts) != fmt.Sprint(expected) {
			t.Errorf("Expected %v, got %v", expected, shorts)
		}
	})
}

func TestGetRandomHex(t *testing.T) {
	t.Run("valid hex16 response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"hex16","length":2,"data":[32767,65535],"success":true}`)
		}))
		defer server.Close()

		client := qrng.NewClientWithAPIKey("---")
		client.APIEndpoint = server.URL

		hexVals, err := client.GetRandomHex(2, 4, "hex16")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		expected := []string{"7fff", "ffff"}
		if fmt.Sprint(hexVals) != fmt.Sprint(expected) {
			t.Errorf("Expected %v, got %v", expected, hexVals)
		}
	})
}

func TestGetRandomNumber(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"type":"uint8","length":1,"data":[127],"success":true}`)
		}))
		defer server.Close()

		client := qrng.NewClient()
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
		client := qrng.NewClient()
		_, err := client.GetRandomNumber(10, 5)
		if err != qrng.ErrInvalidRange {
			t.Errorf("Expected ErrInvalidRange, got %v", err)
		}
	})
}

func TestClientConfiguration(t *testing.T) {
	t.Run("custom HTTP client", func(t *testing.T) {
		client := qrng.NewClient()
		client.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	})

	t.Run("API key client", func(t *testing.T) {
		client := qrng.NewClientWithAPIKey("---")
		if client.APIKey != "---" {
			t.Error("API key not set properly")
		}
	})
}

func TestAPIKeyHandling(t *testing.T) {
	t.Run("missing API key for secured endpoint", func(t *testing.T) {
		client := qrng.NewClientWithAPIKey("")
		client.APIEndpoint = "https://api.quantumnumbers.anu.edu.au"
		_, err := client.GetRandomUint16(1)
		if err == nil || !errors.Is(err, qrng.ErrMissingAPIKey) {
			t.Errorf("Expected missing API key error, got %v", err)
		}
	})

	t.Run("API key header inclusion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-api-key") != "---" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprintln(w, `{"type":"uint16","length":1,"data":[42],"success":true}`)
		}))
		defer server.Close()

		client := qrng.NewClientWithAPIKey("---")
		client.APIEndpoint = server.URL

		_, err := client.GetRandomUint16(1)
		if err != nil {
			t.Fatalf("Failed with valid API key: %v", err)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("invalid hex type", func(t *testing.T) {
		client := qrng.NewClientWithAPIKey("---")
		_, err := client.GetRandomHex(1, 2, "hex32")
		if err == nil || !errors.Is(err, qrng.ErrInvalidHexType) {
			t.Errorf("Expected invalid hex type error, got %v", err)
		}
	})

	t.Run("invalid block size", func(t *testing.T) {
		client := qrng.NewClientWithAPIKey("---")
		_, err := client.GetRandomHex(1, 11, "hex8")
		if err == nil || !errors.Is(err, qrng.ErrInvalidBlockSize) {
			t.Errorf("Expected invalid block size error, got %v", err)
		}
	})
}
