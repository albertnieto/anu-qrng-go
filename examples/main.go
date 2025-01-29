package main

import (
	"fmt"
	"log"
	"time"

	qrng "github.com/albertnieto/anu-qrng-go"
)

func main() {
	client := qrng.NewClient()

	// Get 8 random bits
	bits, err := client.GetRandomBits(8)
	if err != nil {
		log.Fatalf("Error getting bits: %v", err)
	}
	fmt.Println("Random bits:", bits)

	// Get 3 random bytes
	bytes, err := client.GetRandomUint8(3)
	if err != nil {
		log.Fatalf("Error getting bytes: %v", err)
	}
	fmt.Println("Random bytes:", bytes)

	// Get a random number between 1 and 100
	num, err := client.GetRandomNumber(1, 100)
	if err != nil {
		log.Fatalf("Error getting number: %v", err)
	}
	fmt.Println("Random number:", num)

	// Be kind to the API
	time.Sleep(1 * time.Second)
}
