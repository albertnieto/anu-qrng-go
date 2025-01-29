# anu-qrng-go

A Go client for Australia National University's quantum random number generator API.  

```go
package main

import (
    "fmt"
    "github.com/yourusername/anu-qrng-go"
)

func main() {
    client := qrng.NewClient()
    
    // Flip 8 quantum coins
    bits, _ := client.GetRandomBits(8) 
    fmt.Println("Quantum bits:", bits) // e.g., [1 0 1 1 0 1 0 1]
    
    // Generate lottery number (1-100)
    num, _ := client.GetRandomNumber(1, 100)
    fmt.Println("Lucky number:", num) // e.g., 57
}
```