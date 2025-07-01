package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math/big"
)

func setupDatabase() {
	var err error
	app.db, err = sql.Open("sqlite3", "./data/local.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS links (
		ID INTEGER PRIMARY KEY,
		short_code TEXT UNIQUE NOT NULL,
		long_link TEXT NOT NULL,
		times_accessed NUMERIC DEFAULT 0
	);
	`

	_, err = app.db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
}

func insertLink(l *Link) (int64, error) {
	// if the short code is not provided, generate one
	if l.ShortCode == "" {
		l.ShortCode = generateShortCode()
	}
	result, err := app.db.Exec(`
	INSERT INTO links (
	                     short_code, long_link
	                     ) VALUES (?, ?)`, l.ShortCode, l.LongLink)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println("Failed to retrieve last insert ID:", err)
		return 0, err
	}

	return id, nil
}

// generateShortCode generates a random 6 character string to be used as a short code
// when one is not provided
func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 6

	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Println("failed to generate random number for short code")
			// fallback to math/rand
			return "fallback"
		}
		shortKey[i] = charset[n.Int64()]
	}
	return string(shortKey)
}

func getLinkByShortLink(shortCode string) (string, error) {
	var l Link
	err := app.db.QueryRow("SELECT id, short_code, long_link, times_accessed FROM links WHERE short_code = ?", shortCode).Scan(&l.ID, &l.ShortCode, &l.LongLink, &l.TimesAccessed)
	if err != nil {
		return "", fmt.Errorf("failed to get link by short code in db query: %w", err)
	}
	err = setVisitNumberToLink(l.ID, l.TimesAccessed+1)
	if err != nil {
		return "", fmt.Errorf("failed to update visit number after getting links: %w", err)
	}
	return l.LongLink, nil
}

func getLinkByID(id int64) (Link, error) {
	var l Link
	err := app.db.QueryRow("SELECT id, short_code, long_link, times_accessed FROM links WHERE id = ?", id).Scan(&l.ID, &l.ShortCode, &l.LongLink, &l.TimesAccessed)
	if err != nil {
		return l, fmt.Errorf("failed to get link by short code in db query: %w", err)
	}
	return l, nil
}

func setVisitNumberToLink(id int, timesAccessed int) error {
	statement := `UPDATE links SET times_accessed = ? WHERE id = ?`
	_, err := app.db.Exec(statement, timesAccessed, id)
	return err
}

func getAllLinks() ([]Link, error) {
	rows, err := app.db.Query("SELECT id, short_code, long_link, times_accessed FROM links")
	if err != nil {
		return nil, err
	}
	var links []Link
	defer rows.Close()

	for rows.Next() {
		var l Link
		err = rows.Scan(&l.ID, &l.ShortCode, &l.LongLink, &l.TimesAccessed)
		if err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}

func deleteLink(shortCode string) error {
	statement := `DELETE FROM links WHERE short_code = ?`
	_, err := app.db.Exec(statement, shortCode)
	return err
}
