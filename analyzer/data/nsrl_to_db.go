package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func resetNSRLTable(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS nsrl_files (
			id SERIAL PRIMARY KEY,
			sha1 VARCHAR(40) UNIQUE,
			md5 VARCHAR(32) UNIQUE,
			sha256 VARCHAR(64) UNIQUE,
			sha512 VARCHAR(128) UNIQUE,
			filesize VARCHAR(128),
			filepath VARCHAR(512)
		);
	`)
	if err != nil {
		log.Fatal(err)
	}
}
func processChunk(chunk [][]string, db *sql.DB, counter int) int {
	chunkLength := len(chunk)
	for _, row := range chunk {
		counter++
		progress := float64(counter) / float64(chunkLength) * 100
		fmt.Printf("Processing: %.2f%%\r", progress)

		if len(row) < 4 {
			continue
		}

		_, err := db.Exec(`
			INSERT INTO nsrl_files (sha1, filesize, filepath)
			VALUES ($1, $2, $3)
			ON CONFLICT (sha1) DO NOTHING;
		`, row[1], row[2], row[3])
		if err != nil {
			log.Println(err)
			continue
		}
	}
	return counter
}
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a full path to data file.")
		return
	}

	filePath := os.Args[1]

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("DB_HOST")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	dbname := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	psqlInfo := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		host, port, dbname, user, password)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	resetNSRLTable(db)
	counter := 0

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(content), "\n")
	chunkSize := 500000

	var wg sync.WaitGroup
	for i := 1; i < len(lines); i += chunkSize {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}
		chunk := make([][]string, end-i)
		for j := i; j < end; j++ {
			chunk[j-i] = strings.Split(lines[j], "\t")
		}
		wg.Add(1)
		go func(chunk [][]string, db *sql.DB, counter int) {
			defer wg.Done()
			counter = processChunk(chunk, db, counter)
		}(chunk, db, counter)
	}
	wg.Wait()

	_, err = db.Exec(`
		CREATE INDEX nsrl_files_md5 ON nsrl_files(md5);
		CREATE INDEX nsrl_files_sha1 ON nsrl_files(sha1);
		CREATE INDEX nsrl_files_sha256 ON nsrl_files(sha256);
		CREATE INDEX nsrl_files_sha512 ON nsrl_files(sha512);
    `)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Processing complete.")
}
