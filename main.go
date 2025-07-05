package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const (
	BrowserFirefox = "firefox"
)

func handle_firefox(db_path *string, domain *string, output io.Writer) int {
	// TODO: copy file to temp location to avoid database lock
	log.Printf("opening %s as a SQLite database...", *db_path)

	db, _ := sql.Open("sqlite3", fmt.Sprintf("file:%s", *db_path))
	defer func() {
		db.Close()
		log.Printf("%s closed.", *db_path)
	}()

	var err error
	var version string
	err = db.QueryRow(`SELECT sqlite_version()`).Scan(&version)
	if err != nil {
		log.Fatalf("error gettin SQLite version: %v", err)
	}
	log.Printf("%s opened. SQLite version: %s", *db_path, version)

	query := `
    SELECT
      host,
      CASE SUBSTR(host, 1, 1) = '.' WHEN 0 THEN 'FALSE' ELSE 'TRUE' END,
      path,
      CASE isSecure WHEN 0 THEN 'FALSE' ELSE 'TRUE' END,
      expiry,
      name,
      value
    FROM
      moz_cookies
  `

	var rows *sql.Rows
	if domain != nil && *domain != "" {
		query += " WHERE host LIKE ?"
		rows, err = db.Query(query, fmt.Sprintf("%%%s", *domain))
	} else {
		rows, err = db.Query(query)
	}

	if err != nil {
		log.Fatalf("error reading cookies: %v", err)
	}

	count := 0
	for rows.Next() {
		var host string
		var includeSubdomains string
		var path string
		var isSecure string
		var expiry string
		var name string
		var value string

		err = rows.Scan(&host, &includeSubdomains, &path, &isSecure, &expiry, &name, &value)
		if err != nil {
			log.Printf("error reading row: %v", err)
		}
		fmt.Fprintf(output, "%s %s %s %s %s %s %s\n", host, includeSubdomains, path, isSecure, expiry, name, value)
		count += 1
	}
	return count
}

func main() {
	db_path := flag.String("db", "", "path to cookies database")
	browser := flag.String("browser", "", "browser, one of: firefox")
	domain := flag.String("domain", "", "domain filter (ends with)")
	output_path := flag.String("output", "cookies.txt", "output path")
	flag.Parse()

	db_file, err := os.Stat(*db_path)
	if err != nil {
		log.Fatalf("error accessing cookies database: %v", err)
	}

	if !db_file.Mode().IsRegular() {
		log.Fatalf("%s is not a regular file", *db_path)
	}

	_, err = os.Stat(*output_path)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("%s already exists, refusing to overwrite.", *output_path)
	}

	output_file, err := os.Create(*output_path)
	if err != nil {
		log.Fatalf("failed to create %s: %v", *output_path, err)
	}
	defer output_file.Close()

	var cookies int

	switch *browser {
	case BrowserFirefox:
		cookies = handle_firefox(db_path, domain, output_file)
	default:
		log.Fatalf("Unsupported browser: %s", *browser)
	}

	log.Printf("written %d cookies to %s", cookies, *output_path)
}
