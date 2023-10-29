package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ScannedFiles struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     string `json:"size"`
	Owner    string `json:"owner"`
	Perm     string `json:"perm"`
	Accessed string `json:"accessed"`
	Created  string `json:"created"`
	Group    string `json:"group"`
	Modified string `json:"modified"`
	MD5      string `json:"MD5"`
	SHA1     string `json:"SHA1"`
	SHA256   string `json:"SHA256"`
	SHA512   string `json:"SHA512"`
}

type ScanRequest struct {
	Files       []ScannedFiles `json:"files"`
	IPv4Address string         `json:"ipv4"`
}

func checkHashes(files *[]ScannedFiles, db *sql.DB) (*[]ScannedFiles, *[]ScannedFiles, error) {
	var validFiles []ScannedFiles
	var invalidFiles []ScannedFiles

	// Prepare the SQL statement for querying the database
	stmt, err := db.Prepare(`
		SELECT MD5, SHA1, SHA256, SHA512
		FROM (
			SELECT MD5, SHA1, SHA256, SHA512 FROM nist
			UNION ALL
			SELECT MD5, SHA1, SHA256, SHA512 FROM verified
		) AS t
		WHERE MD5 = $1 OR SHA1 = $2 OR SHA256 = $3 OR SHA512 = $4
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare SQL statement: %v", err)
	}

	// Prepare the SQL statement for checking if an identical entry exists in the "candidates" table
	checkStmt, err := db.Prepare(`
		SELECT COUNT(*) FROM candidates
		WHERE MD5 = $1 AND SHA1 = $2 AND SHA256 = $3 AND SHA512 = $4
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare SQL statement: %v", err)
	}

	for _, file := range *files {
		// Execute the SQL statement with the hash values from the file
		rows, err := stmt.Query(file.MD5, file.SHA1, file.SHA256, file.SHA512)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query database: %v", err)
		}

		var existingHashes []ScannedFiles
		for rows.Next() {
			var scannedFile ScannedFiles
			err := rows.Scan(
				&scannedFile.MD5,
				&scannedFile.SHA1,
				&scannedFile.SHA256,
				&scannedFile.SHA512,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to scan row: %v", err)
			}
			existingHashes = append(existingHashes, scannedFile)
		}
		rows.Close()

		if len(existingHashes) > 0 {
			// At least one hash exists in the database, update the entries with all hash values
			for _, existingFile := range existingHashes {
				// Update the entry with all hash values if the respective hash value is empty
				if existingFile.MD5 == "" && file.MD5 != "" {
					existingFile.MD5 = file.MD5
				}
				if existingFile.SHA1 == "" && file.SHA1 != "" {
					existingFile.SHA1 = file.SHA1
				}
				if existingFile.SHA256 == "" && file.SHA256 != "" {
					existingFile.SHA256 = file.SHA256
				}
				if existingFile.SHA512 == "" && file.SHA512 != "" {
					existingFile.SHA512 = file.SHA512
				}

				// Update the entry in the nist table with the updated hash values
				_, err = db.Exec(`
					UPDATE nist
					SET MD5 = $1, SHA1 = $2, SHA256 = $3, SHA512 = $4
					WHERE Name = $5
				`, existingFile.MD5, existingFile.SHA1, existingFile.SHA256, existingFile.SHA512, existingFile.Name)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to update nist table: %v", err)
				}

				// Update the entry in the verified table with the updated hash values
				_, err = db.Exec(`
					UPDATE verified
					SET MD5 = $1, SHA1 = $2, SHA256 = $3, SHA512 = $4
					WHERE Name = $5
				`, existingFile.MD5, existingFile.SHA1, existingFile.SHA256, existingFile.SHA512, existingFile.Name)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to update verified table: %v", err)
				}

				// Add the updated file to the validFiles list
				validFiles = append(validFiles, existingFile)
			}
		} else {
			// No hashes from the file exist in the database, check if an identical entry exists in the "candidates" table
			var count int
			err = checkStmt.QueryRow(file.MD5, file.SHA1, file.SHA256, file.SHA512).Scan(&count)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to query database: %v", err)
			}

			if count == 0 {
				// No identical entry exists in the "candidates" table, add the file to the table
				_, err = db.Exec(`
					INSERT INTO candidates (MD5, SHA1, SHA256, SHA512, filesize, filepath)
					VALUES ($1, $2, $3, $4, $5)
				`, file.MD5, file.SHA1, file.SHA256, file.SHA512, file.Size, file.Path)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to insert into candidates table: %v", err)
				}
			}

			// Add the file to the invalidFiles list
			invalidFiles = append(invalidFiles, file)
		}
	}
	// Index tables
	indexTables(db)

	return &validFiles, &invalidFiles, nil
}

func indexTables(db *sql.DB) {
	_, err := db.Exec(`
		CREATE INDEX nsrl_md5 ON nsrl(md5);
		CREATE INDEX nsrl_sha1 ON nsrl(sha1);
		CREATE INDEX nsrl_sha256 ON nsrl(sha256);
		CREATE INDEX nsrl_sha512 ON nsrl(sha512);

		CREATE INDEX verified_md5 ON verified(md5);
		CREATE INDEX verified_sha1 ON verified(sha1);
		CREATE INDEX verified_sha256 ON verified(sha256);
		CREATE INDEX verified_sha512 ON verified(sha512);

		CREATE INDEX candidates_md5 ON candidates(md5);
		CREATE INDEX candidates_sha1 ON candidates(sha1);
		CREATE INDEX candidates_sha256 ON candidates(sha256);
		CREATE INDEX candidates_sha512 ON candidates(sha512);
    `)
	if err != nil {
		log.Fatal(err)
	}
}

func prepareReport(scanData *ScanRequest, validFiles *[]ScannedFiles, invalidFiles *[]ScannedFiles) {
	// does some stuff

}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("DB_HOST")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	dbname := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	// Creates connection with the database
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

	var scanData ScanRequest

	// Parses json data and writes it to scanData var
	err = json.NewDecoder(os.Stdin).Decode(&scanData)
	if err != nil {
		log.Fatal("Failed to decode JSON data:", err)
	}

	/////////////////////////////
	// For debuging --- REMOVE //
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(10000)
	filename := fmt.Sprintf("/home/netsec/Desktop/praktika/service/analyzer/output_%d.json", randomNum)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Failed to create output file:", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(scanData)
	if err != nil {
		log.Fatal("Failed to write JSON content to file:", err)
	}
	// For debuging --- REMOVE //
	/////////////////////////////

	// process data

	validFiles, invalidFiles, err := checkHashes(&scanData.Files, db)
	if err != nil {
		log.Println("Database query failed:", err)
	}

	prepareReport(&scanData, validFiles, invalidFiles)
}
