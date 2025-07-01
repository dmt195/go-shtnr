package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
)

var testApp App

func TestMain(m *testing.M) {
	// Setup: Create a temporary database file for testing
	tempDBFile := "./test_local.sqlite"
	var openErr error
	testApp.db, openErr = sql.Open("sqlite3", tempDBFile)
	if openErr != nil {
		fmt.Printf("Error opening database: %v\n", openErr)
		os.Exit(1)
	}
	// Initialize the global app variable for testing
	app = &App{
		config: &Config{
			defaultAdminUser:     "testadmin",
			defaultAdminPassword: "testpassword",
			isDevelopment:        true,
			apiKey:               "testapikey",
		},
	}

	// Hash the admin password for the test app
	var hashedPassword []byte
	var err error
	hashedPassword, err = bcrypt.GenerateFromPassword([]byte(app.config.defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Failed to hash password for test app: %v\n", err)
		os.Exit(1)
	}
	app.hashPassword = string(hashedPassword)

	// Initialize securecookie for the test app
	hashKey := securecookie.GenerateRandomKey(32)
	blockKey := securecookie.GenerateRandomKey(32)
	app.sc = securecookie.New(hashKey, blockKey)

	app.db = testApp.db

	// Initialize the database schema
	createTable := `
	CREATE TABLE IF NOT EXISTS links (
		ID INTEGER PRIMARY KEY,
		short_code TEXT UNIQUE NOT NULL,
		long_link TEXT NOT NULL,
		times_accessed NUMERIC DEFAULT 0
	);
	`
	_, err = testApp.db.Exec(createTable)
	if err != nil {
		fmt.Printf("Failed to create test database table: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown: Close the database and remove the temporary file
	testApp.db.Close()
	os.Remove(tempDBFile)

	os.Exit(code)
}

func clearTable() {
	testApp.db.Exec("DELETE FROM links")
}

func TestInsertLink(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "testshort", LongLink: "http://example.com"}
	id, err := insertLink(&link)
	if err != nil {
		t.Fatalf("insertLink failed: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID, got 0")
	}

	// Test auto-generation of short code
	link2 := Link{LongLink: "http://example.org"}
	id2, err := insertLink(&link2)
	if err != nil {
		t.Fatalf("insertLink with auto-gen failed: %v", err)
	}
	if id2 == 0 {
		t.Error("Expected non-zero ID for auto-gen, got 0")
	}
	if link2.ShortCode == "" {
		t.Error("Expected short code to be generated, got empty")
	}
}

func TestGetLinkByShortLink(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "gettest", LongLink: "http://get.com"}
	_, _ = insertLink(&link)

	longURL, err := getLinkByShortLink("gettest")
	if err != nil {
		t.Fatalf("getLinkByShortLink failed: %v", err)
	}
	if longURL != link.LongLink {
		t.Errorf("Expected %s, got %s", link.LongLink, longURL)
	}

	// Test non-existent short link
	_, err = getLinkByShortLink("nonexistent")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Expected sql.ErrNoRows for non-existent link, got %v", err)
	}
}

func TestGetLinkByID(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "idtest", LongLink: "http://id.com"}
	id, _ := insertLink(&link)

	retrievedLink, err := getLinkByID(id)
	if err != nil {
		t.Fatalf("getLinkByID failed: %v", err)
	}
	if retrievedLink.ID != int(id) || retrievedLink.ShortCode != link.ShortCode || retrievedLink.LongLink != link.LongLink {
		t.Errorf("Retrieved link mismatch: %+v", retrievedLink)
	}

	// Test non-existent ID
	_, err = getLinkByID(9999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Expected sql.ErrNoRows for non-existent ID, got %v", err)
	}
}

func TestSetVisitNumberToLink(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "visitest", LongLink: "http://visit.com"}
	id, _ := insertLink(&link)

	err := setVisitNumberToLink(int(id), 10)
	if err != nil {
		t.Fatalf("setVisitNumberToLink failed: %v", err)
	}

	retrievedLink, _ := getLinkByID(id)
	if retrievedLink.TimesAccessed != 10 {
		t.Errorf("Expected TimesAccessed to be 10, got %d", retrievedLink.TimesAccessed)
	}
}

func TestGetAllLinks(t *testing.T) {
	clearTable()
	_, _ = insertLink(&Link{ShortCode: "all1", LongLink: "http://all1.com"})
	_, _ = insertLink(&Link{ShortCode: "all2", LongLink: "http://all2.com"})

	links, err := getAllLinks()
	if err != nil {
		t.Fatalf("getAllLinks failed: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}
}

func TestDeleteLink(t *testing.T) {
	clearTable()
	link := Link{ShortCode: "deltest", LongLink: "http://del.com"}
	_, _ = insertLink(&link)

	err := deleteLink("deltest")
	if err != nil {
		t.Fatalf("deleteLink failed: %v", err)
	}

	_, err = getLinkByShortLink("deltest")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Error("Expected link to be deleted, but it was found")
	}
}

func TestGenerateShortCode(t *testing.T) {
	code := generateShortCode()
	if len(code) != 6 {
		t.Errorf("Expected short code length 6, got %d", len(code))
	}
	// Check if it contains only allowed characters (basic check)
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range code {
		found := false
		for _, c := range charset {
			if char == c {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Short code contains invalid character: %c", char)
		}
	}
}
