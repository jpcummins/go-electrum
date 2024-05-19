package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run example/*.go <command>")
		os.Exit(1)
	}
	cmd := os.Args[1]

	switch cmd {
	case "server":
		TestServer()
	case "get-transaction":
		if len(os.Args) > 2 {
			GetTransaction(os.Args[2])
		} else {
			GetTransaction("")
		}
	case "get-detailed-transaction":
		if len(os.Args) > 2 {
			GetDetailedTransaction(os.Args[2])
		} else {
			GetDetailedTransaction("")
		}
	case "get-detailed-address-history":
		if len(os.Args) > 2 {
			GetDetailedAddressHistory(os.Args[2])
		} else {
			GetDetailedAddressHistory("")
		}
	case "test":
		fmt.Println("test")
	}
}
