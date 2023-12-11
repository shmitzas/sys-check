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
			INSERT INTO files (sha1, filesize, filepath, status)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (sha1) DO NOTHING;
		`, row[1], row[2], row[3], "verified")
		if err != nil {
			log.Println(err)
			continue
		}
	}
	return counter
}
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./nsrl_to_db <full path to sanitized data file>")
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

	fmt.Println("Processing complete.")
}
