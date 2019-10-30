package main

import (
	"flag"
	"fmt"
	"log"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nlopes/slack"
) 

func mustHaveToken(tokenFile string) string {
	info, err := os.Stat(tokenFile)
	if err != nil {
		log.Fatalf("Error stating tokenfile %s %s", tokenFile, err)
	}

	if (info.Mode().Perm() & 0077) != 0 {
		log.Fatalf("Tokenfile %s is accessible to group or world with perms %s, exiting...", tokenFile, info.Mode().Perm())
	}
	
    dat, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		log.Fatalf("Error reading tokenfile %s %s", tokenFile, err)
	}
	
	return strings.TrimSpace(string(dat))
}

func mustAuth(token string) *slack.Client {
	api := slack.New(token)

	_, err := api.AuthTest()
	if err != nil {
		log.Fatalf("Erroring authing to slack: %s", err)
	}

	return api
}

func main() {
	tokenFile := flag.String("tokenfile", "tokenfile.txt", "The file containing your slack token")
	flag.Parse()

	token := mustHaveToken(*tokenFile)
	api := mustAuth(token)

	// channels, err := api.GetChannels(false)
	// if err != nil {
	// 	fmt.Printf("%s\n", err)
	// 	return
	// }
	// for _, channel := range channels {
	// 	fmt.Println(channel.Name)
	// 	// channel is of type conversation & groupConversation
	// 	// see all available methods in `conversation.go`
	// }

	// ims, err := api.GetIMChannels()
	// if err != nil {
	// 	fmt.Printf("%s\n", err)
	// 	return
	// }

	// latest appears to be not set
	// for _, im := range ims {
	// 	fmt.Println(im.Latest)
		// channel is of type conversation & groupConversation
		// see all available methods in `conversation.go`
		hist, _ := api.GetIMHistory("DFUBKMT7V", slack.NewHistoryParameters())
		for _, msg := range hist.Messages {
			fmt.Println(msg.Timestamp)
		}
	
	
}
