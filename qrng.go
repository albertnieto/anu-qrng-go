package qrng

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// Maximum number of uint8 values we can request per API call
	maxUint8Length = 1024
	// Maximum number of bits we can retrieve (maxUint8Length * 8 bits per byte)
	maxBits = maxUint8Length * 8
)

var (
	// When min > max in GetRandomNumber
	ErrInvalidRange = errors.New("min cannot be greater than max")
	// When the requested range exceeds API capabilities
	ErrRangeTooLarge = errors.New("range size exceeds maximum supported value")
)

// QRNGClient is a client for the ANU QRNG API.
// The zero value is not usable, use NewClient to create a properly initialized client.
type QRNGClient struct {
	// Defaults to "https://qrng.anu.edu.au/API/jsonI.php".
	APIEndpoint string

	// Underlying HTTP client used to make requests.
	// If nil, a default client with a 10-second timeout is used.
	HTTPClient *http.Client
}

// NewClient creates a new QRNGClient with default settings.
// The default HTTP client has a 10-second timeout.
func NewClient() *QRNGClient {
	return &QRNGClient{
		APIEndpoint: "https://qrng.anu.edu.au/API/jsonI.php",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// QRNGResponse represents the structure of the API response.
type QRNGResponse struct {
	Type           string   `json:"type"`           // Type of data returned
	Length         int      `json:"length"`         // Number of data items returned
	Success        bool     `json:"success"`        // Indicates if the request was successful
	Data           []int    `json:"data"`           // The random numbers requested
	CompletionTime string   `json:"completionTime"` // Timestamp of request completion
	Seed           string   `json:"seed"`           // Seed used for random number generation
	Refresh        bool     `json:"refresh"`        // Indicates if seed was refreshed
	Error          string   `json:"error"`          // Error message if success is false
	Info           []string `json:"info"`           // Additional information about the request
}

// GetRandomBits retrieves a slice of random bits (0s and 1s) from the QRNG API.
// numBits specifies the number of bits to retrieve, which must be between 1 and 8192.
// The bits are extracted from uint8 values returned by the API, starting from the most significant bit.
// Returns an error if the request fails, the API returns an error, or numBits is out of range.
func (c *QRNGClient) GetRandomBits(numBits int) ([]int, error) {
	if numBits < 1 || numBits > maxBits {
		return nil, fmt.Errorf("numBits must be between 1 and %d", maxBits)
	}

	requiredBytes := (numBits + 7) / 8
	qr, err := c.doRequest(requiredBytes, "uint8")
	if err != nil {
		return nil, err
	}

	bits := make([]int, 0, numBits)
	for _, byteVal := range qr.Data {
		for i := 7; i >= 0; i-- {
			bit := (byteVal >> i) & 1
			bits = append(bits, int(bit))
			if len(bits) == numBits {
				break
			}
		}
		if len(bits) == numBits {
			break
		}
	}

	return bits, nil
}

// GetRandomUint8 fetches random unsigned 8-bit integers from the QRNG API.
// numBytes specifies the number of bytes to retrieve (1-1024).
// Returns an error if the request fails, the API returns an error, or numBytes is out of range.
func (c *QRNGClient) GetRandomUint8(numBytes int) ([]uint8, error) {
	if numBytes < 1 || numBytes > maxUint8Length {
		return nil, fmt.Errorf("numBytes must be between 1 and %d", maxUint8Length)
	}

	qr, err := c.doRequest(numBytes, "uint8")
	if err != nil {
		return nil, err
	}

	randomBytes := make([]uint8, len(qr.Data))
	for i, val := range qr.Data {
		if val < 0 || val > 255 {
			return nil, fmt.Errorf("invalid byte value %d at index %d", val, i)
		}
		randomBytes[i] = uint8(val)
	}

	return randomBytes, nil
}

// GetRandomNumber returns a uniformly distributed random integer in the range [min, max].
// The range must satisfy min <= max. For optimal performance, the range size (max - min + 1)
// should be less than 2^8192. Uses rejection sampling to ensure uniform distribution.
func (c *QRNGClient) GetRandomNumber(min, max int) (int, error) {
	if min > max {
		return 0, ErrInvalidRange
	}

	rangeSize := max - min + 1
	if rangeSize <= 0 { // Handle integer overflow
		return 0, ErrRangeTooLarge
	}

	bitSize := 1
	for (1 << bitSize) < rangeSize {
		bitSize++
	}

	if bitSize > maxBits {
		return 0, fmt.Errorf("%w: maximum supported bits is %d", ErrRangeTooLarge, maxBits)
	}

	requiredBytes := (bitSize + 7) / 8
	mask := (1 << bitSize) - 1

	for {
		randomBytes, err := c.GetRandomUint8(requiredBytes)
		if err != nil {
			return 0, err
		}

		var randInt int
		for _, b := range randomBytes {
			randInt = (randInt << 8) | int(b)
		}
		randInt &= mask

		if randInt < rangeSize {
			return min + randInt, nil
		}
	}
}

// doRequest handles the common logic for making API requests and parsing responses
func (c *QRNGClient) doRequest(length int, dataType string) (*QRNGResponse, error) {
	params := url.Values{}
	params.Add("length", strconv.Itoa(length))
	params.Add("type", dataType)

	reqURL := c.APIEndpoint + "?" + params.Encode()

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var qr QRNGResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !qr.Success {
		if qr.Error != "" {
			return nil, fmt.Errorf("api error: %s", qr.Error)
		}
		return nil, errors.New("api request failed without specific error")
	}

	if len(qr.Data) < length {
		return nil, fmt.Errorf("insufficient data returned: expected %d, got %d", length, len(qr.Data))
	}

	return &qr, nil
}
