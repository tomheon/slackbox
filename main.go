package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func mustHaveToken(tokenPath string) string {
	info, err := os.Stat(tokenPath)
	if err != nil {
		log.Fatalf("Error stating tokenpath %s %s", tokenPath, err)
	}

	if (info.Mode().Perm() & 0077) != 0 {
		log.Fatalf("Tokenpath %s is accessible to group or world with perms %s, exiting...", tokenPath, info.Mode().Perm())
	}

	dat, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		log.Fatalf("Error reading tokenpath %s %s", tokenPath, err)
	}

	return strings.TrimSpace(string(dat))
}

func mustConnectAPI(token string) *SlackBoxAPI {
	api, err := ConnectAPI(token)

	if err != nil {
		log.Fatalf("Erroring connecting to slack: %s", err)
	}

	return api
}

func mustConnectDB(dbPath string) *SlackBoxDB {
	db, err := ConnectDB(dbPath)

	if err != nil {
		log.Fatalf("Erroring connect to db at %s: %s", dbPath, err)
	}

	return db
}

func main() {
	tokenPath := flag.String("tokenpath", "tokenfile.txt", "The path containing your slack token")
	dbPath := flag.String("dbpath", "slackbox.db", "The path to the message db")
	flag.Parse()

	token := mustHaveToken(*tokenPath)
	api := mustConnectAPI(token)
	db := mustConnectDB(*dbPath)

	conversations, err := api.FetchConversations()
	if err != nil {
		log.Fatal(err)
	}

	err = db.UpdateConversations(conversations)
	if err != nil {
		log.Fatal(err)
	}

	unackedConversations, err := db.GetUnackedConversations()
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range unackedConversations {
		fmt.Println(c)
	}
}
