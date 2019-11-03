package main

import (
	"flag"
	"fmt"
	// "io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/rivo/tview"
	"github.com/gdamore/tcell"
	"github.com/pkg/browser"
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

func createSelectFunc(api *SlackBoxAPI, ac AcknowledgedConversation, list *tview.List, app *tview.Application) (func()) {
	return func() {
		ts := ac.GetBestLinkableTs()
		id := ac.ID
		link, err := api.FetchConversationLink(id, ts)
		if err == nil {
			err = browser.OpenURL(link)
		}
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

func main() {
	tokenPath := flag.String("tokenpath", "tokenfile.txt", "The path containing your slack token")
	dbPath := flag.String("dbpath", "slackbox.db", "The path to the message db")
	flag.Parse()

	// TODO see if we can check perms on api, reject if we have too many
	token := mustHaveToken(*tokenPath)
	api := mustConnectAPI(token)
	db := mustConnectDB(*dbPath)

	app := tview.NewApplication()
	list := tview.NewList()
	
	unackedConversations, err := updateAndFindUnacked(api, db)
	if err != nil {
		log.Fatalf("%s", err)
	}

	for _, uc := range unackedConversations {
		list.AddItem(fmt.Sprintf("[::b]%s", uc.DisplayName), "", 0, createSelectFunc(api, uc, list, app))
	}

	list.ShowSecondaryText(false)
	list.SetDoneFunc(func() {
		app.Stop()
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()

		if key == tcell.KeyRune {
			ch := event.Rune()
			switch ch {
			case 'j':
				event = tcell.NewEventKey(tcell.KeyDown, ch, event.Modifiers())
			case 'k':
				event = tcell.NewEventKey(tcell.KeyUp, ch, event.Modifiers())
			case 'q':
				app.Stop()
				event = nil
			}
		}
		
		return event
	})
	
	if err := app.SetRoot(list, true).Run(); err != nil {
		panic(err)
	}
}

// type RlContext struct {
// 	unackedConversations []AcknowledgedConversation
// 	pageSize             int
// 	curPage              int
// 	curIdx               int
// 	acked                map[int]bool

// 	db  *SlackBoxDB
// 	api *SlackBoxAPI
// }

// func rlLoop(api *SlackBoxAPI, db *SlackBoxDB) error {
// 	rl, err := readline.New("\033[31m»\033[0m ")
// 	if err != nil {
// 		return err
// 	}

// 	unackedConversations, err := updateAndFindUnacked(api, db)
// 	if err != nil {
// 		return err
// 	}

// 	rlContext := &RlContext{db: db, api: api, unackedConversations: unackedConversations, curPage: 0, curIdx: 0, pageSize: 2, acked: make(map[int]bool)}

// 	defer rl.Close()

// 	err = doList(rlContext)

// 	for {
// 		line, rlerr := rl.Readline()
// 		if rlerr == readline.ErrInterrupt {
// 			if len(line) == 0 {
// 				break
// 			} else {
// 				continue
// 			}
// 		} else if rlerr == io.EOF {
// 			break
// 		}

// 		line = strings.TrimSpace(line)

// 		switch {
// 		case strings.HasPrefix(line, "l"):
// 			err = doList(rlContext)
// 		case strings.HasPrefix(line, "g"):
// 			err = doGo(rlContext, -1)
// 		case strings.HasPrefix(line, "a"):
// 			err = doAck(rlContext, -1)
// 		}

// 		if err != nil {
// 			return err
// 		}

// 		rl.Refresh()
// 	}

// 	return nil
// }

// func doGo(rlContext *RlContext, userSel int) error {
// 	uc := rlContext.GetCurrentConveration()
// 	ts := uc.GetBestLinkableTs()
// 	id := uc.ID
// 	link, err := rlContext.api.FetchConversationLink(id, ts)
// 	if err != nil {
// 		return err
// 	}
// 	return browser.OpenURL(link)
// }

// func doAck(rlContext *RlContext, userSel int) error {
// 	uc := rlContext.GetCurrentConveration()
// 	id := uc.ID
// 	ts := uc.LatestMsgTs

// 	err := rlContext.db.AckConversation(id, ts)
// 	if err != nil {
// 		return err
// 	}

// 	rlContext.MarkAcked(rlContext.curIdx)
// 	err = doNext(rlContext)
// 	if err != nil {
// 		return err
// 	}
// 	err = doList(rlContext)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func doNext(rlContext *RlContext) error {
// 	rlContext.Next()
// 	return nil
// }

// func (rlContext *RlContext) Next() {
// 	_, endPageIdx := rlContext.CalcPageIdxs()
// 	if rlContext.curIdx+1 >= endPageIdx {
// 		rlContext.NextPage()
// 	} else {
// 		rlContext.curIdx++
// 	}
// }

// func (rlContext *RlContext) NextPage() {
// 	startPageIdx, endPageIdx := rlContext.CalcPageIdxs()
// 	if endPageIdx < len(rlContext.unackedConversations) {
// 		rlContext.curPage++
// 		startPageIdx, _ = rlContext.CalcPageIdxs()
// 		rlContext.curIdx = startPageIdx
// 	}
// }

// func (rlContext *RlContext) MarkAcked(idx int) {
// 	rlContext.acked[idx] = true
// }

// func (rlContext *RlContext) CalcPageIdxs() (int, int) {
// 	startPageIdx := rlContext.curPage * rlContext.pageSize
// 	endPageIdx := startPageIdx + rlContext.pageSize
// 	if endPageIdx >= len(rlContext.unackedConversations) {
// 		endPageIdx = len(rlContext.unackedConversations)
// 	}

// 	return startPageIdx, endPageIdx
// }

// func (rlContext *RlContext) GetCurrentConveration() AcknowledgedConversation {
// 	return rlContext.unackedConversations[rlContext.curIdx]
// }

// func doList(rlContext *RlContext) error {
// 	startPageIdx, endPageIdx := rlContext.CalcPageIdxs()

// 	for i, uc := range rlContext.unackedConversations[startPageIdx:endPageIdx] {
// 		unacked := " + "
// 		_, ok := rlContext.acked[startPageIdx+i]
// 		if ok {
// 			unacked = "   "
// 		}
// 		curSel := " "
// 		if rlContext.curIdx == startPageIdx+i {
// 			curSel = "*"
// 		}
// 		fmt.Printf("%d: %s%s%s\n", i+1, curSel, unacked, uc.DisplayName)
// 	}

// 	return nil
// }
