package main

import (
	"io/ioutil"
	"os"
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
