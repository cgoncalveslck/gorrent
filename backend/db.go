package backend

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB() {
    var err error
    db, err = sql.Open("sqlite3", "./gorrent.db")
    if err != nil {
        log.Fatalf("Error opening database: %v", err)
    }

    createTable()
}

func createTable() {
    createTableSQL := `CREATE TABLE IF NOT EXISTS torrents (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
        "name" TEXT,
        "size" INTEGER,
        "status" TEXT
    );`

    statement, err := db.Prepare(createTableSQL)
    if err != nil {
        log.Fatalf("Error preparing table creation statement: %v", err)
    }
    _, err = statement.Exec()
    if err != nil {
        log.Fatalf("Error executing table creation statement: %v", err)
    }
}

func AddTorrent(t *Torrent) {
    insertSQL := `INSERT INTO torrents (name, size, status) VALUES (?, ?, ?)`
    statement, err := db.Prepare(insertSQL)
    if err != nil {
        log.Fatalf("Error preparing insert statement: %v", err)
    }
    _, err = statement.Exec(t.TorrentName, t.TotalLength, t.Status)
    if err != nil {
        log.Fatalf("Error executing insert statement: %v", err)
    }
}

func GetTorrents() ([]Torrent, error) {
    rows, err := db.Query("SELECT id, name, size, status FROM torrents")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var torrents []Torrent
    for rows.Next() {
        var torrent Torrent
        err = rows.Scan(&torrent.ID, &torrent.TorrentName, &torrent.TotalLength, &torrent.Status)
        if err != nil {
            return nil, err
        }
        torrents = append(torrents, torrent)
    }
    return torrents, nil
}