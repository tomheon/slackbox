package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Simple struct to hide the sqlite3 details.
type SlackBoxDB struct {
	db *sql.DB
}

func (db *SlackBoxDB) UpdateConversation(conversation Conversation) error {
	sql := `
      insert into conversations 
        (id, conversation_type, display_name, latest_msg_ts)
      values
        (?,  ?,                 ?,            ?)
      on conflict (id)
      do update set
      display_name = excluded.display_name,
      latest_msg_ts = excluded.latest_msg_ts
      where excluded.latest_msg_ts > latest_msg_ts
        `
	stmt, err := db.db.Prepare(sql)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(conversation.ID, conversation.ConversationType, conversation.DisplayName, conversation.LatestMsgTs)
	if err != nil {
		return err
	}

	return nil
}

func (db *SlackBoxDB) UpdateConversations(conversations []Conversation) error {
	for _, conversation := range conversations {
		err := db.UpdateConversation(conversation)
		if err != nil {
			return err
		}
	}

	return nil
}

func ConnectDB(dbPath string) (*SlackBoxDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	err = initialize(db)
	if err != nil {
		return nil, err
	}

	return &SlackBoxDB{db}, nil
}

// Initialize the db, creating the schema if necessary.  This function is
// idempotent, and a db may be safely initialized multiple times..
func initialize(db *sql.DB) error {
	schemaSql := `
      -- the list of conversations we're tracking
      create table if not exists conversations (
        -- we use the im/channel id directly from the slack api, which is text
        id text not null primary key,
        -- either 'im' or 'channel'
        conversation_type text not null,
        display_name text not null,
        -- the slack api uses text timestamps
        latest_msg_ts text
      );

      create table if not exists acknowledgements (
        conversation_id text not null,
        -- slack ts indicating that conversation has been acknowledged up to 
        -- and including this msg
        acknowledged_through_ts text not null,
        -- seconds since the epoch, db time when ack was made (*not* slack ts)
        acknowledged_at int
      );

      create index if not exists ack_convo_idx on acknowledgements (
        conversation_id, acknowledged_through_ts);
	`
	_, err := db.Exec(schemaSql)
	return err
}
