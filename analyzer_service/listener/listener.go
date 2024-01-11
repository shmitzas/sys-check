package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"

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
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Failed to get the current user:", err)
		os.Exit(1)
	}
	envPath := fmt.Sprintf("/home/%s/.sys-check/.env/listener.env", currentUser.Username)
	err = godotenv.Load(envPath)
	if err != nil {
		log.Fatal("Error loading .env file")
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

	tmpFile, err := ioutil.TempFile("", "sys_check-request-jsondata-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

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
	errorLogDir := os.Getenv("ERROR_LOGS")
	dirErr := os.MkdirAll(errorLogDir, 0755)
	if dirErr != nil {
		log.Println("failed to create logs directory:", dirErr)
		return
	}
	filepath := fmt.Sprintf("%s/error.log", errorLogDir)
	logFile, fileErr := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if fileErr != nil {
		log.Println("failed to open log file:", fileErr)
		return
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.Println("error occurred during request processing:", err)
}
