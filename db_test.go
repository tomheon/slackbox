package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func bumpVersion(t *testing.T, db *SlackBoxDB) {
	bumpVersionSql := `
    update version set version = version + 1
    `
	_, err := db.db.Exec(bumpVersionSql)
	if err != nil {
		t.Fatalf("Could not bump version of db %s", err)
	}
}

func TestErrOnUnsupportedVersion(t *testing.T) {
	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Could not create tempfile %s", err)
	}

	defer os.Remove(tempfile.Name())

	db, err := ConnectDB(tempfile.Name())
	if err != nil {
		t.Fatalf("Could not connect to test db %s", err)
	}

	// just prove that we can close the db and reconnect without error, so
	// we're really testing the version bump
	db.db.Close()

	db, err = ConnectDB(tempfile.Name())
	if err != nil {
		t.Fatalf("Could not connect to test db after close / reopen %s", err)
	}

	bumpVersion(t, db)

	db.db.Close()

	db, err = ConnectDB(tempfile.Name())
	if _, ok := err.(*unsupportedVersionError); !ok {
		t.Errorf("Expected version error, got %s", err)
	}
}

func TestUpdateConversationNew(t *testing.T) {
	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Could not create tempfile %s", err)
	}

	defer os.Remove(tempfile.Name())

	db, err := ConnectDB(tempfile.Name())
	c := Conversation{ID: "someconvo", ConversationType: "im", DisplayName: "display", LatestMsgTs: "1.0000"}

	_, found, err := db.GetConversation(c.ID)
	if found {
		t.Fatal("Found conversation before it existed")
	}
	if err != nil {
		t.Fatalf("Error trying to find conversation %s", err)
	}

	err = db.UpdateConversation(c)
	if err != nil {
		t.Errorf("Error trying to update conversation %s", err)
	}

	foundC, found, err := db.GetConversation(c.ID)
	if !found {
		t.Error("Couldn't find conversation post update")
	}
	if err != nil {
		t.Errorf("Error trying to find conversation %s", err)
	}
	if !reflect.DeepEqual(c, foundC) {
		t.Errorf("Expected to find conversation %s, found %s", c, foundC)
	}
}

func TestUpdateConversationLater(t *testing.T) {
	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Could not create tempfile %s", err)
	}

	defer os.Remove(tempfile.Name())

	db, err := ConnectDB(tempfile.Name())
	c := Conversation{ID: "someconvo", ConversationType: "im", DisplayName: "display", LatestMsgTs: "1.0000"}
	err = db.UpdateConversation(c)
	if err != nil {
		t.Errorf("Error trying to update conversation %s", err)
	}

	foundC, found, err := db.GetConversation(c.ID)
	if !found {
		t.Error("Couldn't find conversation post update")
	}
	if err != nil {
		t.Errorf("Error trying to find conversation %s", err)
	}
	if !reflect.DeepEqual(c, foundC) {
		t.Errorf("Expected to find conversation %s, found %s", c, foundC)
	}

	c2 := Conversation{ID: "someconvo", ConversationType: "channel", DisplayName: "display2", LatestMsgTs: "2.0000"}

	err = db.UpdateConversation(c2)
	if err != nil {
		t.Errorf("Error trying to update conversation %s", err)
	}

	foundC, found, err = db.GetConversation(c.ID)

	if !found {
		t.Error("Couldn't find conversation post update")
	}

	if err != nil {
		t.Errorf("Error trying to find conversation %s", err)
	}

	if c2.LatestMsgTs != foundC.LatestMsgTs {
		t.Errorf("Didn't update timestamp %s %s", c2, foundC)
	}

	if c2.DisplayName != foundC.DisplayName {
		t.Errorf("didn't update displayname %s %s", c2, foundC)
	}

	if c2.ConversationType == foundC.ConversationType {
		t.Errorf("mistakenly updated conversation type %s %s", c2, foundC)
	}
}

// TODO test update existing with same ts
// TODO test update existing with earlier ts
// TODO test updateconversations
