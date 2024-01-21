package main

import (
	// import necessary packages
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

var childProcesses []*exec.Cmd

const NumTranscations = 100

func performTaskOne() {
	defer cleanupChildProcesses()

	// Start the wallet application
	go startProcess("wallet", "8080")

	// Start  5 nodes
	for i := 1; i <= 5; i++ {
		port := 3100 + i
		time.Sleep(1 * time.Second)
		go startProcess("node", fmt.Sprintf("%d", port))
	}

	go startProcess("consensus", "1111")

	// Ensure nodes are up and running
	time.Sleep(10 * time.Second)

	// Create 100 transactions
	createAndBroadcastTransaction()

	log.Println("All transcation sent, Now wait 2 mins to complete...")

	time.Sleep(20 * time.Minute)

	log.Println("Completed 100 blocks")

	cleanupChildProcesses()
}

func startProcess(mode string, port string) {
	cmd := exec.Command("go", "run", ".", mode, port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start %s on port %s: %v", mode, port, err)
	} else {
		childProcesses = append(childProcesses, cmd)
	}
}

// simulateRandomTransactions generates 100 random transactions between users.
func simulateRandomTransactions(users []string) []*Transaction {
	var transactions []*Transaction

	if len(users) < 2 {
		log.Println("Not enough users to create transactions")
		return transactions
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate transactions where each user sends to the next user in the list
	for i := 0; i < NumTranscations; i++ {
		senderIndex := i % len(users)         // Loop through the users
		receiverIndex := (i + 1) % len(users) // Next user in the list

		sender := users[senderIndex]
		receiver := users[receiverIndex]
		amount := 1 + r.Intn(2) // Random amount between 1 and 2

		tx := NewTransaction(sender, receiver, amount)
		transactions = append(transactions, tx)
	}

	return transactions
}

func readUsersFromFile(filename string) ([]string, error) {
	var users []string

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) > 2 {
			users = append(users, parts[2]) // Assuming the address is the third part
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// createAndBroadcastTransaction creates and broadcasts transactions to all nodes.
func createAndBroadcastTransaction() {

	// Read 5 users from users.txt
	users, err := readUsersFromFile("users.txt")
	if err != nil {
		log.Fatalf("Failed to read users: %v", err)
	}

	// Generate a list of transactions
	transactions := simulateRandomTransactions(users)

	// Loop through each transaction and broadcast it
	for _, tx := range transactions {
		BroadcastTransactionToNodes(tx)
		time.Sleep(20 * time.Second) // Sleep for 0.5 seconds
	}
}

func cleanupChildProcesses() {
	for _, cmd := range childProcesses {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				log.Printf("Failed to kill process: %s", err)
			}
		}
	}
}
