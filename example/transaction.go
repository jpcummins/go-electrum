package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/jpcummins/go-electrum/electrum"
)

func GetTransaction(txid string) {
	if txid == "" {
		txid = "66555dfb0f823623caae5ac27dc1458a78a1cfe36ab85792a05583453446d9e2"
	}

	ctx := context.Background()
	client, err := electrum.NewClientSSL(
		ctx,
		"electrum.bitaroo.net:50002",
		&tls.Config{
			InsecureSkipVerify: true,
		},
		electrum.WithTimeout(time.Second*10),
	)
	if err != nil {
		panic(err)
	}
	client.ServerVersion(ctx, "2.7.11", "1.4.2")

	// Get transaction
	tx, err := client.GetTransaction(ctx, txid)
	if err != nil {
		log.Fatalf("GetTransaction err: %v\n", err)
	}

	fmt.Printf("tx: %v\n", tx)
}
