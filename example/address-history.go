package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jpcummins/go-electrum/electrum"
)

func GetDetailedAddressHistory(address string) {
	if address == "" {
		address = "3AwUscZWWWqgkEg3t4Xb9kY6c281KDwHW2"
	}

	ctx := context.Background()
	client, err := electrum.NewClientSSL(
		ctx,
		//"electrum.bitaroo.net:50002",
		"ru.poiuty.com:50002",
		&tls.Config{
			InsecureSkipVerify: true,
		},
		electrum.WithTimeout(time.Second*10),
	)
	if err != nil {
		panic(err)
	}
	client.ServerVersion(ctx, "2.7.11", "1.4.2")

	scriptHash, err := electrum.AddressToElectrumScriptHash(address)
	if err != nil {
		log.Fatalf("AddressToElectrumScriptHash err: %v\n", err)
	}

	// get script hash history
	history, err := client.GetHistory(ctx, scriptHash)
	if err != nil {
		log.Fatalf("GetHistory err: %v\n", err)
	}

	dHistory, err := client.DetailHistory(ctx, address, history)
	if err != nil {
		log.Fatalf("DetailHistory err: %v\n", err)
	}

	x, _ := json.Marshal(dHistory)
	fmt.Printf("%s", x)
}
