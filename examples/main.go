package main

import (
	"fmt"
	"log"
	"time"

	qrng "github.com/albertnieto/anu-qrng-go"
)

func main() {
	// Test authenticated client with API key
	runClientTests("Authenticated Client", qrng.NewClientWithAPIKey("---"))

	// Test legacy client without API key
	runClientTests("Legacy Client", qrng.NewClient())

	time.Sleep(1 * time.Second)
}

func runClientTests(clientName string, client *qrng.QRNGClient) {
	fmt.Printf("\n--- Testing %s ---\n", clientName)

	// Get 8 random bits
	bits, err := client.GetRandomBits(8)
	if err != nil {
		log.Printf("[%s] Error getting bits: %v", clientName, err)
	} else {
		fmt.Printf("[%s] Random bits: %v\n", clientName, bits)
	}

	// Get 3 random bytes
	bytes, err := client.GetRandomUint8(3)
	if err != nil {
		log.Printf("[%s] Error getting bytes: %v", clientName, err)
	} else {
		fmt.Printf("[%s] Random bytes: %v\n", clientName, bytes)
	}

	// Get a random number between 1 and 100
	num, err := client.GetRandomNumber(1, 100)
	if err != nil {
		log.Printf("[%s] Error getting number: %v", clientName, err)
	} else {
		fmt.Printf("[%s] Random number: %d\n", clientName, num)
	}

	time.Sleep(1 * time.Second)
}
