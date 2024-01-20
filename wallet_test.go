package main

import (
	"testing"
)

func TestWalletAddress(t *testing.T) {
	wallet := NewWallet()
	address := wallet.Address()

	if len(address) == 0 {
		t.Error("Address() failed, the address should not be empty")
	}
}

func TestHashPubKey(t *testing.T) {
	wallet := NewWallet()
	hash := HashPubKey(wallet.PublicKey)

	if len(hash) == 0 {
		t.Error("HashPubKey() failed, the hash should not be empty")
	}
}
