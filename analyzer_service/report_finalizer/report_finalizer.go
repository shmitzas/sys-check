package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Hostname    string `json:"hostname"`
}

func main() {
	// Get metadata from command-line arguments
	var metadata Metadata
	if len(os.Args) != 2 {
		fmt.Println("Usage: bash report_finalizer <ipv4_address>")
		return
	}

	metadata.IPv4Address = os.Args[1]

	// Directory path and filename pattern
	dirPath := fmt.Sprintf("/tmp/sys-check/reports-%s", metadata.IPv4Address)

	// Find all JSON files matching the pattern
	filePaths, err := findJSONFiles(dirPath)
	if err != nil {
		fmt.Printf("Error finding JSON files: %v\n", err)
		fmt.Println("Usage: bash report_finalizer <ipv4_address>")
		return
	}

	// Read and combine the JSON contents
	var combinedReport Report
	for _, filePath := range filePaths {
		report, err := readJSONFile(filePath)
		if err != nil {
			fmt.Printf("Error reading JSON file %s: %v\n", filePath, err)
			continue
		}
		combinedReport.VerifiedFiles = append(combinedReport.VerifiedFiles, report.VerifiedFiles...)
		combinedReport.CandidateFiles = append(combinedReport.CandidateFiles, report.CandidateFiles...)
		combinedReport.MaliciousFiles = append(combinedReport.MaliciousFiles, report.MaliciousFiles...)
		combinedReport.MaliciousVars = append(combinedReport.MaliciousVars, report.MaliciousVars...)
	}
	combinedReport.Metadata = metadata

	RemoveReports(filePaths)

	// Write the combined report to a new JSON file
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

// findJSONFiles finds all JSON files in the given directory that match the filename pattern
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

// readJSONFile reads and parses the JSON file
func readJSONFile(filePath string) (Report, error) {
	var report Report

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return report, err
	}
	defer file.Close()

	// Decode the JSON contents
	err = json.NewDecoder(file).Decode(&report)
	if err != nil {
		return report, err
	}

	return report, nil
}

// writeJSONFile writes the JSON data to a file
func writeJSONFile(filePath string, data interface{}) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode the data as JSON and write to the file
	err = json.NewEncoder(file).Encode(data)
	if err != nil {
		return err
	}

	return nil
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
