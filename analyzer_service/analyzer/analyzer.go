package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ScannedFiles struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int    `json:"size"`
	Owner      string `json:"owner"`
	Perm       string `json:"perm"`
	Accessed   string `json:"accessed"`
	Created    string `json:"created"`
	Group      string `json:"group"`
	Modified   string `json:"modified"`
	MD5        string `json:"MD5"`
	SHA1       string `json:"SHA1"`
	SHA256     string `json:"SHA256"`
	SHA512     string `json:"SHA512"`
	FileStatus string `json:"fileStatus"`
}

type Metadata struct {
	IPv4Address string `json:"ip_address"`
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
		fileStatus := checkIfFileExists(&file, db)

		if fileStatus == "verified" {
			file.FileStatus = "verified"
			verifiedFiles = append(verifiedFiles, file)
		}
		if fileStatus == "malicious" {
			file.FileStatus = "malicious"
			maliciousFiles = append(maliciousFiles, file)
		}
		if fileStatus == "candidate" {
			file.FileStatus = "candidate"
			candidateFiles = append(candidateFiles, file)
		}
		if fileStatus == "none" {
			file.FileStatus = "candidate"
			candidateFiles = append(candidateFiles, file)
			err := insertNewFileData(&file, db)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return &verifiedFiles, &maliciousFiles, &candidateFiles, nil
}

func checkIfFileExists(file *ScannedFiles, db *sql.DB) string {
	stmt, err := db.Prepare(`
		SELECT MD5, SHA1, SHA256, SHA512, status
		FROM files
		WHERE MD5 = $1 OR SHA1 = $2 OR SHA256 = $3 OR SHA512 = $4;
	`)
	if err != nil {
		fmt.Errorf("error preparing query: \n%v", err)
		return "none"
	}

	rows, err := stmt.Query(file.MD5, file.SHA1, file.SHA256, file.SHA512)
	if err != nil {
		fmt.Errorf("error executing query: \n%v", err)
		return "none"
	}

	var existingHashes []ScannedFiles
	for rows.Next() {
		var scannedFile ScannedFiles
		err := rows.Scan(
			&scannedFile.MD5,
			&scannedFile.SHA1,
			&scannedFile.SHA256,
			&scannedFile.SHA512,
			&scannedFile.FileStatus,
		)
		if err != nil {
			fmt.Errorf("error checking query results: \n%v", err)
			return "none"
		}
		existingHashes = append(existingHashes, scannedFile)
	}
	rows.Close()
	var result ScannedFiles
	if len(existingHashes) > 0 {
		result = existingHashes[0]
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
		_, err = db.Exec(`
			UPDATE files
			SET MD5 = $1, SHA1 = $2, SHA256 = $3, SHA512 = $4
			WHERE filepath = $5;
		`, result.MD5, result.SHA1, result.SHA256, result.SHA512, result.Path)
		if err != nil {
			fmt.Errorf("error updating entry: \n%v", err)
			return "none"
		}
	}
	if result.FileStatus == "" {
		return "none"
	}
	return result.FileStatus
}

func insertNewFileData(file *ScannedFiles, db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO files (MD5, SHA1, SHA256, SHA512, filesize, filepath, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`, file.MD5, file.SHA1, file.SHA256, file.SHA512, file.Size, file.Path, "candidate")
	if err != nil {
		return fmt.Errorf("failed to insert new file data into files table: \n%v\nFile details:\nPath: %v\nSize: %v\nMD5: %v\nSHA1: %v\nSHA256: %v\nSHA512: %v", err, file.Path, file.Size, file.MD5, file.SHA1, file.SHA256, file.SHA512)
	}
	return nil
}

func saveReport(scanMetadata *Metadata, verifiedFiles *[]ScannedFiles, maliciousFiles *[]ScannedFiles, candidateFiles *[]ScannedFiles, maliciousVars *[]string) {

	reportsDir := os.Getenv("REPORTS_DIR")
	var report Report
	report.Metadata = *scanMetadata
	report.VerifiedFiles = *verifiedFiles
	report.CandidateFiles = *candidateFiles
	report.MaliciousFiles = *maliciousFiles
	report.MaliciousVars = *maliciousVars
	directory := fmt.Sprintf("%s-%s", reportsDir, scanMetadata.IPv4Address)

	err := os.MkdirAll(directory, 0755)

	if err != nil {
		fmt.Printf("error creating directory '%s': %v\n", directory, err)
		return
	}

	currentTime := time.Now()
	timestamp := currentTime.Format("2006-01-02-15:04:05")
	filnename := fmt.Sprintf("%s/report-%s.json", directory, timestamp)

	file, err := os.Create(filnename)
	if err != nil {
		fmt.Println("error creating file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(report)
	if err != nil {
		fmt.Println("error encoding JSON:", err)
		return
	}
}

func readJson() (*ScanRequest, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("please provide a full path to data file")
	}

	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var request ScanRequest

	err = json.Unmarshal(data, &request)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return &request, nil
}

func main() {
	envFilePath := "/etc/sys_check/analyzer.env"
	err := godotenv.Load(envFilePath)
	if err != nil {
		log.Fatal("error loading .env file:", err)
	}

	host := os.Getenv("DB_HOST")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	dbName := os.Getenv("DB_NAME")
	dbSchema := os.Getenv("DB_SCHEMA")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	psqlInfo := fmt.Sprintf("host=%s port=%d dbname=%s search_path=%s user=%s password=%s sslmode=disable",
		host, port, dbName, dbSchema, user, password)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	scanData, err := readJson()
	if err != nil {
		log.Fatal(err)
	}

	batchSize := 1000
	batches := split_to_batches(scanData.Files, batchSize)

	var wg sync.WaitGroup
	wg.Add(len(batches))

	for _, batch := range batches {
		go process_batch(&batch, &scanData.Metadata, db, &wg)
	}

	wg.Wait()
}

func split_to_batches(files []ScannedFiles, batchSize int) [][]ScannedFiles {
	var result [][]ScannedFiles

	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		result = append(result, files[i:end])
	}

	return result
}

func process_batch(files *[]ScannedFiles, metadata *Metadata, db *sql.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	validatedData, maliciousVars, err := validateData(*files)
	if err != nil {
		log.Println("data validation failed:", err)
	}

	verifiedFiles, maliciousFiles, candidateFiles, err := checkHashes(validatedData, db)
	if err != nil {
		log.Println("database query failed:", err)
	}

	saveReport(metadata, verifiedFiles, maliciousFiles, candidateFiles, maliciousVars)
}

func validateData(files []ScannedFiles) (*[]ScannedFiles, *[]string, error) {
	injectionPattern := `(?i)[^a-z0-9\s](['";\\/\-*])`

	regexpPattern, err := regexp.Compile(injectionPattern)
	if err != nil {
		fmt.Printf("error compiling regex pattern: %s\n", err)
		return &files, nil, err
	}

	maliciousVars := make([]string, 0)

	for i := 0; i < len(files); i++ {
		if matched := regexpPattern.MatchString(files[i].MD5); matched {
			maliciousVars = append(maliciousVars, files[i].MD5)
			files = append(files[:i], files[i+1])
		}
		if matched := regexpPattern.MatchString(files[i].SHA1); matched {
			maliciousVars = append(maliciousVars, files[i].SHA1)
			files = append(files[:i], files[i+1])
		}
		if matched := regexpPattern.MatchString(files[i].SHA256); matched {
			maliciousVars = append(maliciousVars, files[i].SHA256)
			files = append(files[:i], files[i+1])
		}
		if matched := regexpPattern.MatchString(files[i].SHA512); matched {
			maliciousVars = append(maliciousVars, files[i].SHA512)
			files = append(files[:i], files[i+1])
		}
	}
	return &files, &maliciousVars, nil
}
