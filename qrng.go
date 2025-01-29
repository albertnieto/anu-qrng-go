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
	"strings"
	"time"
)

const (
	maxUint8Length  = 1024
	maxUint16Length = 1024
	maxBits         = maxUint8Length * 8
	defaultTimeout  = 10 * time.Second
)

var (
	ErrInvalidRange     = errors.New("min cannot be greater than max")
	ErrRangeTooLarge    = errors.New("range size exceeds maximum supported value")
	ErrMissingAPIKey    = errors.New("API key required for this endpoint")
	ErrInvalidHexType   = errors.New("invalid hex type, must be hex8 or hex16")
	ErrInvalidBlockSize = errors.New("block size must be between 1-10")
)

type QRNGClient struct {
	APIEndpoint string
	HTTPClient  *http.Client
	APIKey      string
	useAPIKey   bool
}

// NewClient creates client for the legacy API (no key required)
func NewClient() *QRNGClient {
	return &QRNGClient{
		APIEndpoint: "https://qrng.anu.edu.au/API/jsonI.php",
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
		useAPIKey: false,
	}
}

// NewClientWithAPIKey creates client for the new authenticated API
func NewClientWithAPIKey(apiKey string) *QRNGClient {
	return &QRNGClient{
		APIEndpoint: "https://api.quantumnumbers.anu.edu.au",
		APIKey:      apiKey,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
		useAPIKey: true,
	}
}

// Update requiresAPIKey check
func (c *QRNGClient) requiresAPIKey() bool {
	return c.useAPIKey
}

type QRNGResponse struct {
	Type           string   `json:"type"`
	Length         int      `json:"length"`
	Success        bool     `json:"success"`
	Data           []int    `json:"data"`
	CompletionTime string   `json:"completionTime"`
	Seed           string   `json:"seed"`
	Refresh        bool     `json:"refresh"`
	Error          string   `json:"error"`
	Info           []string `json:"info"`
}

func (c *QRNGClient) GetRandomBits(numBits int) ([]int, error) {
	if numBits < 1 || numBits > maxBits {
		return nil, fmt.Errorf("numBits must be between 1 and %d", maxBits)
	}

	requiredBytes := (numBits + 7) / 8
	qr, err := c.doRequest(requiredBytes, "uint8", 0)
	if err != nil {
		return nil, err
	}

	return extractBits(qr.Data, numBits), nil
}

func extractBits(data []int, numBits int) []int {
	bits := make([]int, 0, numBits)
	for _, byteVal := range data {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (byteVal>>i)&1)
			if len(bits) == numBits {
				return bits
			}
		}
	}
	return bits
}

func (c *QRNGClient) GetRandomUint8(numBytes int) ([]uint8, error) {
	if numBytes < 1 || numBytes > maxUint8Length {
		return nil, fmt.Errorf("numBytes must be between 1 and %d", maxUint8Length)
	}

	qr, err := c.doRequest(numBytes, "uint8", 0)
	if err != nil {
		return nil, err
	}

	return convertUint8(qr.Data), nil
}

func convertUint8(data []int) []uint8 {
	result := make([]uint8, len(data))
	for i, v := range data {
		result[i] = uint8(v)
	}
	return result
}

func (c *QRNGClient) GetRandomUint16(numShorts int) ([]uint16, error) {
	if numShorts < 1 || numShorts > maxUint16Length {
		return nil, fmt.Errorf("numShorts must be between 1 and %d", maxUint16Length)
	}

	qr, err := c.doRequest(numShorts, "uint16", 0)
	if err != nil {
		return nil, err
	}

	return convertUint16(qr.Data), nil
}

func convertUint16(data []int) []uint16 {
	result := make([]uint16, len(data))
	for i, v := range data {
		result[i] = uint16(v)
	}
	return result
}

func (c *QRNGClient) GetRandomHex(blockCount, blockSize int, hexType string) ([]string, error) {
	if hexType != "hex8" && hexType != "hex16" {
		return nil, ErrInvalidHexType
	}

	if blockSize < 1 || blockSize > 10 {
		return nil, ErrInvalidBlockSize
	}

	qr, err := c.doRequest(blockCount, hexType, blockSize)
	if err != nil {
		return nil, err
	}

	return formatHex(qr.Data, hexType, blockSize), nil
}

func formatHex(data []int, hexType string, blockSize int) []string {
	result := make([]string, len(data))
	format := "%04x"
	if hexType == "hex8" {
		format = fmt.Sprintf("%%0%dx", blockSize*2)
	}

	for i, v := range data {
		result[i] = fmt.Sprintf(format, v)
	}
	return result
}

func (c *QRNGClient) GetRandomNumber(min, max int) (int, error) {
	if min > max {
		return 0, ErrInvalidRange
	}

	rangeSize := max - min + 1
	if rangeSize <= 0 {
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

		randInt := bytesToInt(randomBytes) & mask
		if randInt < rangeSize {
			return min + randInt, nil
		}
	}
}

func bytesToInt(bytes []uint8) int {
	var result int
	for _, b := range bytes {
		result = (result << 8) | int(b)
	}
	return result
}

func (c *QRNGClient) doRequest(length int, dataType string, blockSize int) (*QRNGResponse, error) {
	if c.requiresAPIKey() && c.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	params := url.Values{
		"length": {strconv.Itoa(length)},
		"type":   {dataType},
	}

	if strings.HasPrefix(dataType, "hex") && blockSize > 0 {
		params.Add("size", strconv.Itoa(blockSize))
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		c.APIEndpoint+"?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	if c.requiresAPIKey() {
		req.Header.Add("x-api-key", c.APIKey)
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, errRead := io.ReadAll(resp.Body)
		if errRead != nil {
			return nil, fmt.Errorf("unexpected status code %d: error reading body: %w", resp.StatusCode, errRead)
		}
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, errBody)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response: %w", err)
	}

	var qr QRNGResponse
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, fmt.Errorf("json parse error: %w", err)
	}

	if !qr.Success {
		if qr.Error != "" {
			return nil, fmt.Errorf("api error: %s", qr.Error)
		}
		return nil, errors.New("api request failed")
	}

	if len(qr.Data) < length {
		return nil, fmt.Errorf("insufficient data: expected %d, got %d", length, len(qr.Data))
	}

	return &qr, nil
}
