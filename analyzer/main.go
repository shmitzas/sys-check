package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type ScannedFiles struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
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

type Scan struct {
	Architecture string         `json:"architecture"`
	Files        []ScannedFiles `json:"files"`
	IPv4Details  struct {
		Address    string `json:"address"`
		Alias      string `json:"alias"`
		Broadcast  string `json:"broadcast"`
		Gateway    string `json:"gateway"`
		Interface  string `json:"interface"`
		MacAddress string `json:"macaddress"`
		MTU        int    `json:"mtu"`
		Netmask    string `json:"netmask"`
		Network    string `json:"network"`
		Prefix     string `json:"prefix"`
		Type       string `json:"type"`
	} `json:"ipv4_details"`
	Kernel string `json:"kernel"`
	OS     string `json:"os"`
}

type Report struct {
	Architecture string         `json:"architecture"`
	ValidFiles   []ScannedFiles `json:"validFiles"`
	InvalidFiles []ScannedFiles `json:"invalidFiles"`
	IPv4Details  struct {
		Address    string `json:"address"`
		Alias      string `json:"alias"`
		Broadcast  string `json:"broadcast"`
		Gateway    string `json:"gateway"`
		Interface  string `json:"interface"`
		MacAddress string `json:"macaddress"`
		MTU        int    `json:"mtu"`
		Netmask    string `json:"netmask"`
		Network    string `json:"network"`
		Prefix     string `json:"prefix"`
		Type       string `json:"type"`
	} `json:"ipv4_details"`
	Kernel string `json:"kernel"`
	OS     string `json:"os"`
}

func readJson() (*Scan, error) {
	// Read the JSON data from the file
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("please provide a full path to data file")
	}

	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Create a variable to store the decoded JSON data
	var scan []Scan

	// Decode the JSON data into the variable
	err = json.Unmarshal(data, &scan)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return &scan[0], nil
}

func compareHashes(scanData Scan, db *sql.DB) (*[]ScannedFiles, *[]ScannedFiles) {
	var validFiles []ScannedFiles
	var invalidFiles []ScannedFiles
	for i := 0; i < len(scanData.Files); i++ {
		query := "SELECT EXISTS(SELECT 1 FROM nsrl_files WHERE sha1 = $1);"
		var exists bool
		err := db.QueryRow(query, scanData.Files[i].SHA1).Scan(&exists)

		if err != nil {
			log.Fatal(err)
		}

		if exists {
			validFiles = append(validFiles, scanData.Files[i])
		} else {
			invalidFiles = append(invalidFiles, scanData.Files[i])
		}
		fmt.Printf("Processing file %d of %d\r", i+1, len(scanData.Files))
	}
	return &validFiles, &invalidFiles
}

func prepareReport(scanData Scan, validFiles []ScannedFiles, invalidFiles []ScannedFiles) {
	reportData := Report{
		Architecture: scanData.Architecture,
		Kernel:       scanData.Kernel,
		OS:           scanData.OS,
		IPv4Details:  scanData.IPv4Details,
		ValidFiles:   validFiles,
		InvalidFiles: invalidFiles,
	}
	// Open the file for writing
	timestamp := time.Now().Format("2006-01-02.15.04.05") // Format the current time as "YYYY-MM-DD.HH.mm.ss"

	// Construct the filename with the timestamp
	filename := fmt.Sprintf("/tmp/sys_check/results/report-%s-%s.json", scanData.IPv4Details.Address, timestamp)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Create a JSON encoder
	encoder := json.NewEncoder(file)

	// Write the ScanData struct to the file
	err = encoder.Encode(reportData)
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

	scanData, err := readJson()
	if err != nil {
		log.Fatal(err)
	}
	validFiles, invalidFiles := compareHashes(*scanData, db)
	prepareReport(*scanData, *validFiles, *invalidFiles)
}
