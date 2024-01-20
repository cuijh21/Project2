package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	mrand "math/rand" // Using alias to avoid conflict
	"time"
)

const (
	version            = byte(0x00) // Version for the wallet address
	addressChecksumLen = 4          // Length of the address checksum
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey // The private key of the wallet
	PublicKey  []byte           // The public key of the wallet
}

// NewWallet creates and returns a new wallet.
func NewWallet() *Wallet {
	private, public := newKeyPair()
	return &Wallet{private, public}
}

// newKeyPair generates a private key and public key pair.
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256() // Using P256 curve for the elliptic curve cryptography
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pubKey
}

// HashPubKey hashes the public key.
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)
	return publicSHA256[:]
}

// Checksum generates a checksum for an address.
func Checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]
}

// Address generates a wallet address.
func (w Wallet) Address() string {
	pubKeyHash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	address := hex.EncodeToString(fullPayload)
	return address
}

// generateRandomCredentials generates random username and password.
func generateRandomCredentials() (string, string) {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	username := fmt.Sprintf("user%d", r.Intn(100000))
	password := fmt.Sprintf("pass%d", r.Intn(100000))
	return username, password
}

// BalanceOf queries the balance of a given address on a given blockchain.
func BalanceOf(bc *Blockchain, address string) int {
	return bc.GetBalance(address)
}
