package main

import (
	"github.com/nlopes/slack"
)

type SlackBoxAPI struct {
	client *slack.Client
}

type Conversation struct {
	ConversationType string
	ID               string
	DisplayName      string
	LatestMsgTs      string
}

func ConnectAPI(token string) (*SlackBoxAPI, error) {
	api := slack.New(token)

	_, err := api.AuthTest()

	return &SlackBoxAPI{api}, err
}

func (api *SlackBoxAPI) recursiveFetchConversations(types []string) ([]slack.Channel, error) {
	ims := make([]slack.Channel, 0)
	params := &slack.GetConversationsParameters{Types: types}

	for {
		newIms, nextCursor, err := api.client.GetConversations(params)

		if err != nil {
			return ims, err
		}

		ims = append(ims, newIms...)

		if nextCursor == "" {
			break
		}

		params.Cursor = nextCursor
	}

	return ims, nil
}

func (api *SlackBoxAPI) FetchConversations() ([]Conversation, error) {
	conversations := make([]Conversation, 0)

	ims, err := api.recursiveFetchConversations([]string{"im"})

	if err != nil {
		return nil, err
	}

	for _, im := range ims {
		conversation, err := api.imToConversation(im.ID, im.User)
		if err != nil {
			return nil, err
		}

		conversations = append(conversations, conversation)
	}

	return conversations, nil
}

func (api *SlackBoxAPI) fetchUserName(imUser string) (string, error) {
	user, err := api.client.GetUserInfo(imUser)
	if err != nil {
		return "", err
	}
	return user.RealName, nil
}

func (api *SlackBoxAPI) imToConversation(imID string, imUser string) (Conversation, error) {
	convo := Conversation{ConversationType: "im", ID: imID}
	userName, err := api.fetchUserName(imUser)
	if err != nil {
		return Conversation{}, err
	}

	convo.DisplayName = userName

	params := &slack.GetConversationHistoryParameters{ChannelID: imID}
	history, err := api.client.GetConversationHistory(params)

	if err != nil {
		return convo, err
	}

	for _, msg := range history.Messages {
		if msg.Timestamp > convo.LatestMsgTs {
			convo.LatestMsgTs = msg.Timestamp
		}
	}

	return convo, nil
}
