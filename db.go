package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const SupportedDBVersion = 1

type AcknowledgedConversation struct {
	Conversation
	AcknowledgedThroughTs string
}

func (a *AcknowledgedConversation) GetBestLinkableTs() string {
	if a.AcknowledgedThroughTs != "" {
		return a.AcknowledgedThroughTs
	}

	return a.LatestMsgTs
}

type unsupportedVersionError struct {
	supportedVersion int
	actualVersion    int
}

func (e *unsupportedVersionError) Error() string {
	return fmt.Sprintf("Actual version %d, supported version %d", e.actualVersion, e.supportedVersion)
}

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

func (db *SlackBoxDB) GetConversation(conversationID string) (Conversation, bool, error) {
	c := Conversation{}

	query := `
    select 
      id, conversation_type, display_name, latest_msg_ts
    from
      conversations
    where
      id = ? 
    `
	rows, err := db.db.Query(query, conversationID)
	if err != nil {
		return c, false, err
	}

	defer rows.Close()

	found := rows.Next()
	if !found {
		return c, false, nil
	}

	err = rows.Scan(&c.ID, &c.ConversationType, &c.DisplayName, &c.LatestMsgTs)
	if err != nil {
		return c, false, err
	}

	found = rows.Next()
	if found {
		return c, false, errors.New("Found duplicate conversation id")
	}

	err = rows.Err()
	if err != nil {
		return c, false, err
	}

	return c, true, nil
}

func (db *SlackBoxDB) AckConversation(id string, ackTs string) error {
	// TODO trim the acks as part of this
	sql := `
      insert into acknowledgements
        (conversation_id, acknowledged_through_ts)
      values
        (?,               ?)
      on conflict(conversation_id, acknowledged_through_ts) do nothing
    `

	_, err := db.db.Exec(sql, id, ackTs)
	return err
}

func (db *SlackBoxDB) UnackConversation(id string, ackTs string) error {
	sql := `
      delete from acknowledgements
      where conversation_id = ? and acknowledged_through_ts = ?
    `
	_, err := db.db.Exec(sql, id, ackTs)
	return err
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

func (db *SlackBoxDB) GetUnackedConversations() ([]AcknowledgedConversation, error) {
	sql := `
      with

      latest_acknowledgements as (
        select
          conversation_id,
          max(acknowledged_through_ts) as acknowledged_through_ts
        from
          acknowledgements
        group by
          conversation_id
      )

      select
        c.id, c.conversation_type, c.display_name, c.latest_msg_ts,
        coalesce(a.acknowledged_through_ts, '')
      from
        conversations c left outer join latest_acknowledgements a
        on c.id = a.conversation_id
      where
        (c.latest_msg_ts > a.acknowledged_through_ts
         or a.acknowledged_through_ts is null)
        -- a blank latest_msg_ts would mean there had never
        -- been a message in the conversation, so we don't
        -- care about it
        and c.latest_msg_ts <> ''
      order by
        c.latest_msg_ts desc,
        c.id asc
    `

	conversations := make([]AcknowledgedConversation, 0)

	rows, err := db.db.Query(sql)
	if err != nil {
		return conversations, err
	}

	defer rows.Close()

	for rows.Next() {
		c := AcknowledgedConversation{}
		err = rows.Scan(&c.ID, &c.ConversationType, &c.DisplayName, &c.LatestMsgTs, &c.AcknowledgedThroughTs)
		if err != nil {
			return conversations, err
		}

		conversations = append(conversations, c)
	}

	if rows.Err() != nil {
		return conversations, rows.Err()
	}

	return conversations, nil
}

func checkSupportedVersion(db *sql.DB) error {
	initVersionSql := `
      create table if not exists version (
        -- singleton should always be 1, regardless of the version,
        -- and lets us maintain a single version row
        singleton int not null primary key,
        version int not null
      );

      insert into version (singleton, version)
      values (1, 1)
      on conflict(singleton) do nothing;
    `

	_, err := db.Exec(initVersionSql)

	if err != nil {
		return err
	}

	rows, err := db.Query("select version from version")
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var version int
		err = rows.Scan(&version)
		if err != nil {
			return err
		}

		if version > SupportedDBVersion {
			return &unsupportedVersionError{actualVersion: version, supportedVersion: SupportedDBVersion}
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

// Initialize the db, creating the schema if necessary.  This function is
// idempotent, and a db may be safely initialized multiple times..
func initialize(db *sql.DB) error {
	err := checkSupportedVersion(db)
	if err != nil {
		return err
	}

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

      create unique index if not exists ack_convo_idx on acknowledgements (
        conversation_id, acknowledged_through_ts);
	`
	_, err = db.Exec(schemaSql)
	return err
}
