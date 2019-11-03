package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/pkg/browser"
	"github.com/rivo/tview"
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

func updateAndFindUnacked(api *SlackBoxAPI, db *SlackBoxDB) ([]AcknowledgedConversation, error) {
	unacked := make([]AcknowledgedConversation, 0)

	conversations, err := api.FetchConversations()
	if err != nil {
		return unacked, err
	}
	err = db.UpdateConversations(conversations)
	if err != nil {
		return unacked, err
	}

	return db.GetUnackedConversations()
}

func createSelectFunc(api *SlackBoxAPI, ac AcknowledgedConversation, list *tview.List, app *tview.Application) func() {
	return func() {
		ts := ac.GetBestLinkableTs()
		id := ac.ID
		link, err := api.FetchConversationLink(id, ts)
		if err == nil {
			err = browser.OpenURL(link)
		}
		// TODO make repeatable modal func
		if err != nil {
			modal := tview.NewModal()
			modal.SetText(fmt.Sprintf("Error: %s", err))
			modal.AddButtons([]string{"OK"})
			modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.SetRoot(list, true)
			})
			app.SetRoot(modal, false)
		}
	}
}

func ackConversation(unackedConversations []AcknowledgedConversation, db *SlackBoxDB, app *tview.Application, list *tview.List) {
	// TODO error modal
	i := list.GetCurrentItem()
	uc := unackedConversations[i]
	id := uc.ID
	ts := uc.LatestMsgTs
	err := db.AckConversation(id, ts)
	if err != nil {
		log.Fatal(err)
	}
	list.SetItemText(i, fmt.Sprintf("  %s", uc.DisplayName), "")
}

func unackConversation(unackedConversations []AcknowledgedConversation, db *SlackBoxDB, app *tview.Application, list *tview.List) {
	// TODO error modal
	i := list.GetCurrentItem()
	uc := unackedConversations[i]
	id := uc.ID
	ts := uc.LatestMsgTs
	err := db.UnackConversation(id, ts)
	if err != nil {
		log.Fatal(err)
	}
	list.SetItemText(i, fmt.Sprintf("[::b]* %s", uc.DisplayName), "")
}

func showHelpModal(app *tview.Application, list *tview.List) {
	modal := tview.NewModal()
	modal.SetText("Navigate with j/k or arrow keys\nr marks a conversation as read\nu marks a conversation as unread again\nEnter opens the current selection in slack\ng re-fetches conversations from slack\nh or ? brings up this help")
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		app.SetRoot(list, true)
	})
	app.SetRoot(modal, false)
}

func createInputCaptureFunc(unackedConversations []AcknowledgedConversation, api *SlackBoxAPI, db *SlackBoxDB, app *tview.Application, list *tview.List) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()

		if key == tcell.KeyRune {
			ch := event.Rune()
			switch ch {
			case 'j':
				event = tcell.NewEventKey(tcell.KeyDown, ch, event.Modifiers())
			case 'k':
				event = tcell.NewEventKey(tcell.KeyUp, ch, event.Modifiers())
			case 'g':
				initList(api, db, app)
				event = nil
			case 'r':
				ackConversation(unackedConversations, db, app, list)
				event = tcell.NewEventKey(tcell.KeyDown, ch, event.Modifiers())
			case 'u':
				unackConversation(unackedConversations, db, app, list)
				event = nil
			case '?':
				showHelpModal(app, list)
				event = nil
			case 'h':
				showHelpModal(app, list)
				event = nil
			case 'q':
				app.Stop()
				event = nil
			}
		}

		return event
	}
}

func initList(api *SlackBoxAPI, db *SlackBoxDB, app *tview.Application) {
	unackedConversations, err := updateAndFindUnacked(api, db)
	if err != nil {
		// TODO make this an error modal
		log.Fatalf("%s", err)
	}

	list := tview.NewList()

	for _, uc := range unackedConversations {
		list.AddItem(fmt.Sprintf("[::b]* %s", uc.DisplayName), "", 0, createSelectFunc(api, uc, list, app))
	}

	list.ShowSecondaryText(false)
	list.SetDoneFunc(func() {
		app.Stop()
	})
	list.SetBorder(true)
	list.SetTitle("Slackbox v1.0 (? or h for help)")

	list.SetInputCapture(createInputCaptureFunc(unackedConversations, api, db, app, list))

	app.SetRoot(list, true)
}

func main() {
	tokenPath := flag.String("tokenpath", "tokenfile.txt", "The path containing your slack token")
	dbPath := flag.String("dbpath", "slackbox.db", "The path to the message db")
	flag.Parse()

	// TODO see if we can check perms on api, reject if we have too many
	token := mustHaveToken(*tokenPath)
	api := mustConnectAPI(token)
	db := mustConnectDB(*dbPath)

	app := tview.NewApplication()
	initList(api, db, app)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
