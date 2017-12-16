package pickup

import (
	"database/sql"
	"log"
)

// PlayerStore is an interface for structs that can store player objects
type PlayerStore interface {
	PlayerExists(id string) bool
	Register(id string, fc string)
	UpdateFriendCode(id string, fc string)
	GetFriendCode(id string) string
	GetPlayer(id string) *Player
}

// SQLitePlayerStore implements PlayerStore and uses a SQLite database to store player data
type SQLitePlayerStore struct {
	DB *sql.DB
}

func (ps SQLitePlayerStore) PlayerExists(id string) bool {
	rows, err := ps.DB.Query("SELECT * FROM Players WHERE DiscordID = ?", id)
	if err != nil {
		log.Print(err)
	}
	return rows.Next()
}

func (ps SQLitePlayerStore) Register(id string, fc string) {
	tx, _ := ps.DB.Begin()
	_, err := ps.DB.Exec("INSERT INTO Players (DiscordID, FriendCode) VALUES (?, ?)", id, fc)
	if err != nil {
		log.Print(err)
	}
	tx.Commit()
}

func (ps SQLitePlayerStore) UpdateFriendCode(id string, fc string) {
	tx, _ := ps.DB.Begin()
	_, err := ps.DB.Exec("UPDATE Players SET FriendCode = ? WHERE DiscordID = ?", fc, id)
	if err != nil {
		log.Print(err)
	}
	tx.Commit()
}

func (ps SQLitePlayerStore) GetFriendCode(id string) string {
	var fc string
	rows, _ := ps.DB.Query("SELECT FriendCode FROM Players WHERE DiscordID = ?", id)
	if rows.Next() {
		rows.Scan(&fc)
	}
	return fc
}

func (ps SQLitePlayerStore) GetPlayer(id string) *Player {
	player := new(Player)
	player.ID = id
	player.FriendCode = ps.GetFriendCode(id)
	return player
}
