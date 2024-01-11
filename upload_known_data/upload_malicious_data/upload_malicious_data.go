package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
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
		fmt.Println("Usage: ./upload_malicious_data <full path to json data file>")
		return
	}

	filePath := os.Args[1]

	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Failed to get the current user:", err)
		os.Exit(1)
	}

	envPath := fmt.Sprintf("/home/%s/.sys-check/.env/upload_data.env", currentUser.Username)
	err = godotenv.Load(envPath)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	host := os.Getenv("DB_HOST")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	dbName := os.Getenv("DB_NAME")
	dbSchema := os.Getenv("DB_SCHEMA")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

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

	fmt.Println("Reading data from file...")

	files, err := readJSONFile(filePath)
	if err != nil {
		log.Fatal("failed to decode JSON data:", err)
	}
	uploadData(files, db)
}

func uploadData(files []ScannedFiles, db *sql.DB) {
	for i, file := range files {
		_, err := db.Exec(`
		INSERT INTO files (MD5, SHA1, SHA256, SHA512, filesize, filepath, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
		`, file.MD5, file.SHA1, file.SHA256, file.SHA512, file.Size, file.Path, "malicious")
		if err != nil {
			fmt.Errorf("failed to insert new file data into files table: %v", err)
		}
		fmt.Printf("\rProgress: %d/%d", i+1, len(files))
	}
}

func readJSONFile(filePath string) ([]ScannedFiles, error) {
	var files []ScannedFiles

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
