package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

// Global variable to store pending transactions.
var pendingTransactions []*Transaction

type Application struct {
	Blockchain   *Blockchain
	PollInterval int // Polling interval in seconds
}

// NewApplication creates a new application instance.
func NewApplication() *Application {
	app := &Application{
		Blockchain:   NewBlockchain(), // Initial load
		PollInterval: 3,               // For example, poll every 3 seconds
	}
	go app.startBlockchainUpdate()
	return app
}

type BlockForTemplate struct {
	Timestamp     int64
	Transactions  []*TransactionForTemplate // Or any transaction info you want to display in the template
	PrevBlockHash string                    // Base64 encoded
	Hash          string                    // Base64 encoded
	Nonce         int
}

type TransactionForTemplate struct {
	ID     string // Hex representation of the hash
	From   string // Sender's address
	To     string // Receiver's address
	Amount int    // Transaction amount
}

func (app *Application) start(port string) {

	// 设置静态文件服务器
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 设置路由
	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/mywallet", app.handleMyWallet)
	http.HandleFunc("/transactions/new", app.handleNewTransaction)
	http.HandleFunc("/blockchain", app.handleViewBlockchain)
	http.HandleFunc("/register", app.handleRegister)
	http.HandleFunc("/login", app.handleLogin)
	http.HandleFunc("/logout", app.handleLogout)
	http.HandleFunc("/transaction-history", app.handleTransactionHistory)

	address := "127.0.0.1:" + port
	log.Printf("Wallet server started on http://127.0.0.1:%s\n", port)
	log.Fatal(http.ListenAndServe(address, nil))

}

// handleIndex handles the request for the home page.
func (app *Application) handleIndex(w http.ResponseWriter, r *http.Request) {
	usernameCookie, err := r.Cookie("username")
	var username string
	if err == nil {
		username = usernameCookie.Value
	}

	data := struct {
		Username string
	}{
		Username: username,
	}

	err = templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleNewTransaction handles the creation of new transactions.
func (app *Application) handleNewTransaction(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("loggedin")
	if err != nil || cookie.Value != "true" {
		// 未登录，重定向到登录页面
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	usernameCookie, err := r.Cookie("username")
	var username string
	if err == nil {
		username = usernameCookie.Value
	}

	// 获取钱包信息
	address, balance, _ := app.getWalletInfo(username)

	data := struct {
		Username string
		Address  string
		Balance  int
	}{
		Username: username,
		Address:  address,
		Balance:  balance,
	}

	if r.Method == "POST" {
		r.ParseForm()
		from := r.FormValue("from")
		to := r.FormValue("to")
		amount, err := strconv.Atoi(r.FormValue("amount"))

		if err != nil {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		tx := NewTransaction(from, to, amount)

		// 将交易添加到挂起列表
		pendingTransactions = append(pendingTransactions, tx)

		// 广播交易到所有已知节点
		BroadcastTransactionToNodes(tx)

		// app.Blockchain.AddTransactionToMempool(tx)
		// app.Blockchain.MineBlock()

		http.Redirect(w, r, "/mywallet", http.StatusSeeOther)
	} else {

		err := templates.ExecuteTemplate(w, "transaction_form.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleViewBlockchain handles the request to view the blockchain.
func (app *Application) handleViewBlockchain(w http.ResponseWriter, r *http.Request) {

	usernameCookie, err := r.Cookie("username")
	var username string
	if err == nil {
		username = usernameCookie.Value
	}

	preparedBlocks := prepareBlocksForTemplate(app.Blockchain.Blocks)

	data := struct {
		Username string
		Blocks   []*BlockForTemplate
	}{
		Username: username,
		Blocks:   preparedBlocks,
	}

	err = templates.ExecuteTemplate(w, "blockchain_view.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleRegister handles the registration request.
func (app *Application) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return
		}

		// new wallet
		wallet := NewWallet()
		address := wallet.Address()

		file, err := os.OpenFile("users.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			http.Error(w, "Unable to register user", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		_, err = file.WriteString(username + ":" + password + ":" + address + "\n")
		if err != nil {
			http.Error(w, "Unable to register user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		err := templates.ExecuteTemplate(w, "register.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleLogin handles the login request.
func (app *Application) handleLogin(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		// 验证用户名和密码
		file, err := os.Open("users.txt")
		if err != nil {
			http.Error(w, "Unable to login", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, ":")
			if parts[0] == username && parts[1] == password {

				// 登录成功，设置cookie
				http.SetCookie(w, &http.Cookie{
					Name:   "loggedin",
					Value:  "true",
					Path:   "/",
					MaxAge: 3600,
				})
				http.SetCookie(w, &http.Cookie{
					Name:   "username",
					Value:  username,
					Path:   "/",
					MaxAge: 3600,
				})

				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		// 登录失败
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
	} else {
		err := templates.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (app *Application) handleLogout(w http.ResponseWriter, r *http.Request) {
	// 设置cookie过期
	http.SetCookie(w, &http.Cookie{
		Name:   "loggedin",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:   "username",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleMyWallet handles the request for the My Wallet page.
func (app *Application) handleMyWallet(w http.ResponseWriter, r *http.Request) {
	usernameCookie, err := r.Cookie("username")
	if err != nil {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}
	username := usernameCookie.Value

	address, balance, transactions := app.getWalletInfo(username)

	data := struct {
		Username     string
		Address      string
		Balance      int
		Transactions []Transaction
	}{
		Username:     username,
		Address:      address,
		Balance:      balance,
		Transactions: transactions,
	}

	err = templates.ExecuteTemplate(w, "mywallet.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getWalletInfo retrieves wallet information based on the username.
func (app *Application) getWalletInfo(username string) (string, int, []Transaction) {
	file, err := os.Open("users.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) == 3 && parts[0] == username {
			address := parts[2]

			balance := BalanceOf(app.Blockchain, address)
			transactions := []Transaction{}
			return address, balance, transactions
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return "", 0, nil
}

func prepareBlocksForTemplate(blocks []*Block) []*BlockForTemplate {
	var blocksForTemplate []*BlockForTemplate
	for _, block := range blocks {
		encodedHash := base64.StdEncoding.EncodeToString(block.Hash)
		encodedPrevHash := base64.StdEncoding.EncodeToString(block.PrevBlockHash)

		preparedTransactions := prepareTransactionsForTemplate(block.Transactions)

		newBlock := &BlockForTemplate{
			Timestamp:     block.Timestamp,
			Transactions:  preparedTransactions,
			PrevBlockHash: encodedPrevHash,
			Hash:          encodedHash,
			Nonce:         block.Nonce,
		}

		blocksForTemplate = append(blocksForTemplate, newBlock)
	}
	return blocksForTemplate
}

func prepareTransactionsForTemplate(transactions []*Transaction) []*TransactionForTemplate {
	var transactionsForTemplate []*TransactionForTemplate
	for _, tx := range transactions {
		txForTemplate := &TransactionForTemplate{
			ID:     fmt.Sprintf("%x", tx.ID),
			From:   tx.From,
			To:     tx.To,
			Amount: tx.Amount,
		}
		transactionsForTemplate = append(transactionsForTemplate, txForTemplate)
	}
	return transactionsForTemplate
}

// ReceiveTransactionConfirmation receives a transaction confirmation.
func ReceiveTransactionConfirmation(txID []byte) {
	for i, tx := range pendingTransactions {
		if bytes.Equal(tx.ID, txID) {
			// 移除已确认的交易
			pendingTransactions = append(pendingTransactions[:i], pendingTransactions[i+1:]...)
			break
		}
	}
}

// BroadcastTransactionToNodes broadcasts a transaction to all known nodes.
func BroadcastTransactionToNodes(tx *Transaction) {
	nodes, err := os.ReadFile("nodes.txt")
	if err != nil {
		log.Printf("Error reading nodes.txt: %v", err)
		return
	}

	for _, node := range strings.Split(string(nodes), "\n") {
		if node != "" {
			go func(node string) {
				client, err := rpc.Dial("tcp", node)
				if err != nil {
					log.Printf("Error dialing node %s: %v", node, err)
					return
				}
				defer client.Close()

				var reply string
				err = client.Call("Node.ReceiveTransaction", tx, &reply)
				if err != nil {
					log.Printf("Error broadcasting transaction to node %s: %v", node, err)
				} else {
					log.Printf("Broadcasted transaction to node %s: %s", node, reply)
				}
			}(node)
		}
	}
}

// startBlockchainUpdate starts the process of updating the blockchain.
func (app *Application) startBlockchainUpdate() {
	ticker := time.NewTicker(time.Duration(app.PollInterval) * time.Second)
	for {
		select {
		case <-ticker.C:
			app.updateBlockchainFromConsensus()
		}
	}
}

// updateBlockchainFromConsensus updates the blockchain from the consensus file.
func (app *Application) updateBlockchainFromConsensus() {
	file, err := os.Open(consensusFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error opening consensus blockchain file: %v", err)
		}
		return
	}
	defer file.Close()

	var blocks []*Block
	err = json.NewDecoder(file).Decode(&blocks)
	if err != nil {
		log.Printf("Error decoding consensus blockchain: %v", err)
		return
	}

	if len(blocks) > 0 {
		app.Blockchain.Blocks = blocks
		log.Println("Blockchain updated from consensus file")
	}
}

// transactionExistsInHistory checks if a transaction already exists in the transaction history.
func transactionExistsInHistory(tx *Transaction, history []*Transaction) bool {
	for _, htx := range history {
		if bytes.Equal(tx.ID, htx.ID) {
			return true
		}
	}
	return false
}

// handleTransactionHistory handles the request for the transaction history page.
func (app *Application) handleTransactionHistory(w http.ResponseWriter, r *http.Request) {
	usernameCookie, err := r.Cookie("username")
	if err != nil {
		http.Error(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}
	username := usernameCookie.Value

	address, _, _ := app.getWalletInfo(username)

	transactions := []*Transaction{}
	for _, block := range app.Blockchain.Blocks {
		for _, tx := range block.Transactions {
			if (tx.From == address || tx.To == address) && !transactionExistsInHistory(tx, transactions) {
				transactions = append(transactions, tx)
			}
		}
	}
	data := struct {
		Username     string
		Transactions []*Transaction
	}{
		Username:     username,
		Transactions: transactions,
	}
	err = templates.ExecuteTemplate(w, "transaction_history.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
