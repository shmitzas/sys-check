package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
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

func main() {
	envFilePath := "/home/hp/Documents/GitHub/sys-check/analyzer_service/listener_service_go/.env"
	err := godotenv.Load(envFilePath)
	if err != nil {
		go logError(fmt.Errorf("error loading .env file: %v", err))
	}

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	address := host + ":" + port
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(address, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	done := make(chan bool)
	errCh := make(chan error)

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	go processRequest(r, done, errCh)

	select {
	case <-done:
		w.WriteHeader(http.StatusOK)
	case err := <-errCh:
		go logError(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func processRequest(r *http.Request, done chan<- bool, errCh chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			errCh <- fmt.Errorf("panic occurred: %v", r)
		}
	}()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errCh <- fmt.Errorf("failed to read request body: %v", err)
		return
	}

	var requestData ScanRequest
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		errCh <- fmt.Errorf("failed to parse JSON data: %v", err)
		return
	}

	if requestData.Status == "processing" {
		err = analyzeData(&requestData)
		if err != nil {
			errCh <- fmt.Errorf("failed to execute analyzer: %v", err)
			return
		}
	}

	if requestData.Status == "final" {
		err = combineReports(&requestData)
		if err != nil {
			errCh <- fmt.Errorf("failed to execute report finalizer: %v", err)
			return
		}
	}

	done <- true
}

func analyzeData(data *ScanRequest) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Save JSON data to a temporary file
	tmpFile, err := ioutil.TempFile("", "sys-check-request-jsondata-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temporary file
	defer tmpFile.Close()

	// Write JSON data to the temporary file
	_, err = tmpFile.Write(jsonData)
	if err != nil {
		return err
	}

	binaryPath := os.Getenv("ANALYZER_BIN")

	cmd := exec.Command(binaryPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Args = append(cmd.Args, tmpFile.Name())

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func combineReports(data *ScanRequest) error {

	binaryPath := os.Getenv("REPORT_FINALIZER_BIN")

	cmd := exec.Command(binaryPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Args = append(cmd.Args, data.Metadata.IPv4Address)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func logError(err error) {
	filepath := "/home/hp/Documents/GitHub/sys-check/analyzer_service/listener_service_go/error.log"
	logFile, fileErr := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if fileErr != nil {
		log.Println("Failed to open log file:", fileErr)
		return
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.Println("Error occurred during request processing:", err)
}
