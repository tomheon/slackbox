package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/nlopes/slack"
)

func main() {
	// todo make this a param, check perms, fail if can be read by anyone but owner
    dat, _ := ioutil.ReadFile("/root/creds.txt")
	token := strings.TrimSpace(string(dat))

	api := slack.New(token)
	// resp, err := api.AuthTest()
	// fmt.Println("hello %s %s", resp, err)

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
