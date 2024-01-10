package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Report struct {
	Metadata       Metadata       `json:"metadata"`
	VerifiedFiles  []ScannedFiles `json:"verifiedFiles"`
	CandidateFiles []ScannedFiles `json:"candidateFiles"`
	MaliciousFiles []ScannedFiles `json:"maliciousFiles"`
	MaliciousVars  []string       `json:"maliciousVariables"`
}

type ScannedFiles struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int    `json:"size"`
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
}

func main() {
	var metadata Metadata
	if len(os.Args) != 2 {
		fmt.Println("missing ipv4_address")
		return
	}

	err := godotenv.Load("/etc/sys_check/report_finalizer.env")
	if err != nil {
		log.Fatal("error loading .env file")
	}

	metadata.IPv4Address = os.Args[1]
	reportsDir := os.Getenv("REPORTS_DIR")
	dirPath := fmt.Sprintf("%s-%s", reportsDir, metadata.IPv4Address)

	filePaths, err := findJSONFiles(dirPath)
	if err != nil {
		fmt.Printf("error finding JSON files: %v\n", err)
		return
	}

	var combinedReport Report
	for _, filePath := range filePaths {
		report, err := readJSONFile(filePath)
		if err != nil {
			fmt.Printf("error reading JSON file %s: %v\n", filePath, err)
			continue
		}
		combinedReport.VerifiedFiles = append(combinedReport.VerifiedFiles, report.VerifiedFiles...)
		combinedReport.CandidateFiles = append(combinedReport.CandidateFiles, report.CandidateFiles...)
		combinedReport.MaliciousFiles = append(combinedReport.MaliciousFiles, report.MaliciousFiles...)
		combinedReport.MaliciousVars = append(combinedReport.MaliciousVars, report.MaliciousVars...)
	}
	combinedReport.Metadata = metadata

	RemoveReports(filePaths)

	WriteFinalReport(dirPath, combinedReport)
}

func findJSONFiles(directory string) ([]string, error) {
	var filePaths []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filePaths = append(filePaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filePaths, nil
}

func readJSONFile(filePath string) (Report, error) {
	var report Report

	file, err := os.Open(filePath)
	if err != nil {
		return report, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&report)
	if err != nil {
		return report, err
	}
	return report, nil
}

func WriteFinalReport(dirPath string, combinedReport Report) {
	file, err := os.Create(fmt.Sprintf("%s/final-report.json", dirPath))
	if err != nil {
		fmt.Println("error creating file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(combinedReport)
	if err != nil {
		fmt.Println("error encoding JSON:", err)
		return
	}
}

func RemoveReports(filePaths []string) error {
	for _, path := range filePaths {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}
	return nil
}
