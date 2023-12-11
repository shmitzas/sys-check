package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

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

type Metadata struct {
	IPv4Address string `json:"ipv4"`
	Hostname    string `json:"hostname"`
}

type ScanRequest struct {
	Files    []ScannedFiles `json:"files"`
	Metadata Metadata       `json:"metadata"`
	Status   string         `json:"status"`
}

type Report struct {
	Metadata       Metadata       `json:"metadata"`
	VerifiedFiles  []ScannedFiles `json:"verifiedFiles"`
	CandidateFiles []ScannedFiles `json:"candidateFiles"`
	MaliciousFiles []ScannedFiles `json:"maliciousFiles"`
	MaliciousVars  []string       `json:"maliciousVariables"`
}

func checkHashes(files *[]ScannedFiles, db *sql.DB) (*[]ScannedFiles, *[]ScannedFiles, *[]ScannedFiles, error) {
	var verifiedFiles []ScannedFiles
	var maliciousFiles []ScannedFiles
	var candidateFiles []ScannedFiles

	for _, file := range *files {

		if checkIfFileExists(&file, db, "verified") {
			verifiedFiles = append(verifiedFiles, file)
		}

		if checkIfFileExists(&file, db, "malicious") {
			maliciousFiles = append(maliciousFiles, file)
		}

		if checkIfFileExists(&file, db, "candidates") {
			candidateFiles = append(candidateFiles, file)
		} else {
			err := insertNewFileData(&file, db)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return &verifiedFiles, &maliciousFiles, &candidateFiles, nil
}

func checkIfFileExists(file *ScannedFiles, db *sql.DB, tableName string) bool {
	// Checking VERIFIED files table
	stmt, err := db.Prepare(`
		SELECT MD5, SHA1, SHA256, SHA512
		FROM ` + tableName + `
		WHERE MD5 = $1 OR SHA1 = $2 OR SHA256 = $3 OR SHA512 = $4
	`)
	if err != nil {
		return false
	}

	rows, err := stmt.Query(file.MD5, file.SHA1, file.SHA256, file.SHA512)
	if err != nil {
		return false
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
			return false
		}
		existingHashes = append(existingHashes, scannedFile)
	}
	rows.Close()

	if len(existingHashes) > 0 {
		var result = existingHashes[0]
		// Update the entry with all hash values if the respective hash value is empty
		if result.MD5 == "" && file.MD5 != "" {
			result.MD5 = file.MD5
		}
		if result.SHA1 == "" && file.SHA1 != "" {
			result.SHA1 = file.SHA1
		}
		if result.SHA256 == "" && file.SHA256 != "" {
			result.SHA256 = file.SHA256
		}
		if result.SHA512 == "" && file.SHA512 != "" {
			result.SHA512 = file.SHA512
		}

		// Update the entry in the files table with the updated hash values
		_, err = db.Exec(`
			UPDATE files
			SET MD5 = $1, SHA1 = $2, SHA256 = $3, SHA512 = $4
			WHERE Name = $5
		`, result.MD5, result.SHA1, result.SHA256, result.SHA512, result.Name)
		if err != nil {
			return true
		}
	}
	return false
}

func insertNewFileData(file *ScannedFiles, db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO files (MD5, SHA1, SHA256, SHA512, filesize, filepath, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, file.MD5, file.SHA1, file.SHA256, file.SHA512, file.Size, file.Path, "candidate")
	if err != nil {
		return fmt.Errorf("failed to insert new file data into files table: %v", err)
	}
	return nil
}

func prepareReport(scanMetadata *Metadata, verifiedFiles *[]ScannedFiles, maliciousFiles *[]ScannedFiles, candidateFiles *[]ScannedFiles, maliciousVars *[]string) {
	var report Report
	report.Metadata = *scanMetadata
	report.VerifiedFiles = *verifiedFiles
	report.CandidateFiles = *candidateFiles
	report.MaliciousFiles = *maliciousFiles
	report.MaliciousVars = *maliciousVars
	directory := "/tmp/sys-check/reports"

	err := os.Mkdir(directory, os.ModeDir)

	if err != nil {
		fmt.Println("Error creating directory:", err)
		return
	}

	filnename := fmt.Sprintf("%s/%s-%s-report.json", directory, scanMetadata.Hostname, scanMetadata.IPv4Address)

	file, err := os.Create(filnename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(report)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

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

	// validate data
	validatedData, maliciousVars, err := validateData(scanData.Files)
	if err != nil {
		log.Println("Data validation failed:", err)
	}

	// process data
	verifiedFiles, maliciousFiles, candidateFiles, err := checkHashes(validatedData, db)
	if err != nil {
		log.Println("Database query failed:", err)
	}

	prepareReport(&scanData.Metadata, verifiedFiles, maliciousFiles, candidateFiles, maliciousVars)
}

func validateData(files []ScannedFiles) (*[]ScannedFiles, *[]string, error) {
	// Regular expression pattern to match symbols that could be used in a SQL injection attack
	injectionPattern := `(?i)[^a-z0-9\s](['";\\/\-*])`

	// Compile the pattern
	regexpPattern, err := regexp.Compile(injectionPattern)
	if err != nil {
		fmt.Printf("Error compiling regex pattern: %s\n", err)
		return &files, nil, err
	}

	// Slice to store malicious variables
	maliciousVars := make([]string, 0)

	for i := 0; i < len(files); i++ {
		// Check MD5
		if matched := regexpPattern.MatchString(files[i].MD5); matched {
			maliciousVars = append(maliciousVars, files[i].MD5)
			files = append(files[:i], files[i+1])
		}

		// Check SHA1
		if matched := regexpPattern.MatchString(files[i].SHA1); matched {
			maliciousVars = append(maliciousVars, files[i].SHA1)
			files = append(files[:i], files[i+1])
		}

		// Check SHA256
		if matched := regexpPattern.MatchString(files[i].SHA256); matched {
			maliciousVars = append(maliciousVars, files[i].SHA256)
			files = append(files[:i], files[i+1])
		}

		// Check SHA512
		if matched := regexpPattern.MatchString(files[i].SHA512); matched {
			maliciousVars = append(maliciousVars, files[i].SHA512)
			files = append(files[:i], files[i+1])
		}

		// Check Size
		if matched := regexpPattern.MatchString(string(files[i].Size)); matched {
			maliciousVars = append(maliciousVars, string(files[i].Size))
			files = append(files[:i], files[i+1])
		}

		// Check Path
		if matched := regexpPattern.MatchString(files[i].Path); matched {
			maliciousVars = append(maliciousVars, files[i].Path)
			files = append(files[:i], files[i+1])
		}
	}
	return &files, &maliciousVars, nil
}
