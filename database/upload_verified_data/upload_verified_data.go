package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ScannedFiles struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	MD5    string `json:"MD5"`
	SHA1   string `json:"SHA1"`
	SHA256 string `json:"SHA256"`
	SHA512 string `json:"SHA512"`
}

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Usage: go run upload_verified_data.go <full path to json data file>")
		return
	}

	filePath := os.Args[1]

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("DB_HOST")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	dbName := os.Getenv("DB_NAME")
	dbSchema := os.Getenv("DB_SCHEMA")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	// Creates connection with the database
	psqlInfo := fmt.Sprintf("host=%s port=%d dbName=%s search_path=%s user=%s password=%s sslmode=disable",
		host, port, dbName, dbSchema, user, password)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	files, err := readJSONFile(filePath)
	if err != nil {
		log.Fatal("Failed to decode JSON data:", err)
	}
	fmt.Println("JSON data loaded.")
	uploadData(files, db)

}

func uploadData(files []ScannedFiles, db *sql.DB) {
	for i, file := range files {
		_, err := db.Exec(`
		INSERT INTO files (MD5, SHA1, SHA256, SHA512, filesize, filepath, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
		`, file.MD5, file.SHA1, file.SHA256, file.SHA512, file.Size, file.Path, "verified")
		if err != nil {
			fmt.Errorf("failed to insert new file data into files table: %v", err)
		}
		// Print progress
		fmt.Printf("\rProgress: %d/%d", i+1, len(files))
	}
}

// readJSONFile reads and parses the JSON file
func readJSONFile(filePath string) ([]ScannedFiles, error) {
	var files []ScannedFiles

	// Open the file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return files, err
	}

	err = json.Unmarshal(file, &files)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return files, nil
}
