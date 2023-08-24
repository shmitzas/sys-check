package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type ScannedFiles struct {
	Accessed string `json:"accessed"`
	Checksum string `json:"checksum"`
	Created  string `json:"created"`
	Group    string `json:"group"`
	Modified string `json:"modified"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Path     string `json:"path"`
	Perm     string `json:"perm"`
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

const (
	host     = "ip"
	port     = port
	dbname   = "dbname"
	user     = "user"
	password = "password"
)

func readJson() (*Scan, error) {
    //This is hardcoded only for testing purposes and is temporary
	data, err := os.ReadFile("/tmp/sys_check/results/scans/scan.json")
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var scan []Scan

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
		err := db.QueryRow(query, scanData.Files[i].Checksum).Scan(&exists)

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
	file, err := os.Create("report-" + scanData.IPv4Details.Address + ".json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(reportData)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
}

func main() {
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
