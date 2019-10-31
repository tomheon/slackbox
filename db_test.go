package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func memoryDB(t *testing.T) *SlackBoxDB {
	db, err := ConnectDB(":memory:")
	if err != nil {
		t.Fatalf("Error creating memory db %s", err)
	}
	return db
}

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
	// Don't use a memory db here so we can re-run initialization
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

func checkUpdate(t *testing.T, db *SlackBoxDB, c Conversation) {
	err := db.UpdateConversation(c)
	if err != nil {
		t.Fatalf("Error trying to update conversation %s", err)
	}
}

func checkGet(t *testing.T, db *SlackBoxDB, id string) Conversation {
	c, found, err := db.GetConversation(id)
	if !found {
		t.Fatalf("Couldn't find conversation post update")
	}
	if err != nil {
		t.Fatalf("Error trying to find conversation %s", err)
	}
	return c
}

func TestUpdateConversationNew(t *testing.T) {
	db := memoryDB(t)

	c := Conversation{ID: "someconvo", ConversationType: "im", DisplayName: "display", LatestMsgTs: "1.0000"}

	_, found, err := db.GetConversation(c.ID)
	if found {
		t.Fatal("Found conversation before it existed")
	}
	if err != nil {
		t.Fatalf("Error trying to find conversation %s", err)
	}

	checkUpdate(t, db, c)
	foundC := checkGet(t, db, c.ID)

	if !reflect.DeepEqual(c, foundC) {
		t.Errorf("Expected to find conversation %s, found %s", c, foundC)
	}
}

func TestUpdateConversationLater(t *testing.T) {
	db := memoryDB(t)

	c := Conversation{ID: "someconvo", ConversationType: "im", DisplayName: "display", LatestMsgTs: "1.0000"}
	checkUpdate(t, db, c)
	foundC := checkGet(t, db, c.ID)

	if !reflect.DeepEqual(c, foundC) {
		t.Errorf("Expected to find conversation %s, found %s", c, foundC)
	}

	c2 := Conversation{ID: "someconvo", ConversationType: "channel", DisplayName: "display2", LatestMsgTs: "2.0000"}

	checkUpdate(t, db, c2)
	foundC = checkGet(t, db, c.ID)

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

func testNoUpdate(t *testing.T, firstTs string, secondTs string) {
	db := memoryDB(t)

	c := Conversation{ID: "someconvo", ConversationType: "im", DisplayName: "display", LatestMsgTs: firstTs}

	checkUpdate(t, db, c)
	foundC := checkGet(t, db, c.ID)

	if !reflect.DeepEqual(c, foundC) {
		t.Errorf("Expected to find conversation %s, found %s", c, foundC)
	}

	c2 := Conversation{ID: "someconvo", ConversationType: "channel", DisplayName: "display2", LatestMsgTs: secondTs}

	checkUpdate(t, db, c2)
	foundC = checkGet(t, db, c.ID)

	if !reflect.DeepEqual(c, foundC) {
		t.Errorf("Expected to find conversation %s, found %s", c, foundC)
	}
	if reflect.DeepEqual(c2, foundC) {
		t.Errorf("Updated conversation mistakenly ts %s %s", c2, foundC)
	}
}

func TestUpdateConversationEarlier(t *testing.T) {
	testNoUpdate(t, "1.000", "0.0000")
}

func TestUpdateConversationSame(t *testing.T) {
	// we shouldn't update on the same timestamp
	testNoUpdate(t, "1.000", "1.000")
}

// TODO test updateconversations
