package electrum

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// AddressToElectrumScriptHash converts valid bitcoin address to electrum scriptHash sha256 encoded, reversed and encoded in hex
// https://electrumx.readthedocs.io/en/latest/protocol-basics.html#script-hashes
func AddressToElectrumScriptHash(addressStr string) (string, error) {
	address, err := btcutil.DecodeAddress(addressStr, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}

	hashSum := sha256.Sum256(script)

	for i, j := 0, len(hashSum)-1; i < j; i, j = i+1, j-1 {
		hashSum[i], hashSum[j] = hashSum[j], hashSum[i]
	}

	return hex.EncodeToString(hashSum[:]), nil
}

// GetTotalSentAndReceived returns the total sent and received for a scripthash.
func GetTotalSentAndReceived(
	address string,
	history []*DetailedMempoolResult,
) (float64, float64) {
	var totalSent, totalReceived float64
	for _, tx := range history {
		if tx.Incoming {
			findAddressFunc[Vout](
				address,
				tx.Vout,
				func(elem Vout, index int) bool {
					totalReceived += elem.Value
					return true
				},
			)
		} else {
			findAddressFunc[VinWithPrevout](
				address,
				tx.Vin,
				func(elem VinWithPrevout, index int) bool {
					totalSent += elem.Prevout.Value
					return true
				},
			)
		}
	}

	return totalSent, totalReceived
}
