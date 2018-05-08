package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const maxUploadSize = 32 << 18
const uploadPath = "./upload"
const defaultAddr = ""
const defaultPort = 8080

var defaultWebhookURL = "http://127.0.0.1:10001/wh/upload/notify"

// UploadError Type for error responces
type UploadError struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type UploadResultFile struct {
	OrigFn string `json:"orig_fn"`
	Fn     string `json:"fn"`
	Size   int64  `json:"size"`
}
type UploadResult struct {
	Files []UploadResultFile `json:"files"`
}

type ResultWrapper struct {
	Code    int
	Message string
	File    UploadResultFile
}

func main() {
	// params
	bindAddr := flag.String("addr", defaultAddr, "host addr to bund")
	bindPort := flag.Int("port", defaultPort, "listen port")
	whParam := flag.String("wh", defaultWebhookURL, "webhook to")
	flag.Parse()
	// envs
	webhookURL := func() string {
		whEnv := os.Getenv("WEBHOOK")
		if whEnv != "" {
			return whEnv
		}
		return *whParam
	}()

	listen := fmt.Sprintf("%s:%d", *bindAddr, *bindPort)
	log.Printf("Server started on %s, use /upload for uploading files. Max file size %d", listen, maxUploadSize)
	log.Printf("Webhook to %s", webhookURL)
	http.HandleFunc("/upload", uploadHandler(webhookURL))
	log.Fatal(http.ListenAndServe(listen, nil))

}

func uploadFileHandler(r *http.Request, key string) (ResultWrapper, error) {

	file, header, err := r.FormFile(key)
	origFn := header.Filename
	if err != nil {
		return ResultWrapper{http.StatusBadRequest, "INVALID_FILE", UploadResultFile{}}, err
	}

	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return ResultWrapper{http.StatusBadRequest, "INVALID_FILE", UploadResultFile{}}, err
	}

	fileName := randToken(12)
	newPath := filepath.Join(uploadPath, fileName)
	fmt.Printf("Form field key %s; Orig file name %s; File: %s\n", key, origFn, newPath)

	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		return ResultWrapper{http.StatusInternalServerError, "CANT_WRITE_FILE", UploadResultFile{}}, err
	}
	defer newFile.Close()
	if _, err := newFile.Write(fileBytes); err != nil {
		return ResultWrapper{http.StatusInternalServerError, "CANT_WRITE_FILE", UploadResultFile{}}, err
	}
	return ResultWrapper{http.StatusOK, "OK", UploadResultFile{origFn, fileName, header.Size}}, nil

}

func uploadHandler(whURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate file size
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			renderError(w, "FILE_SIZE_EXCEED", http.StatusBadRequest)
			log.Println(err)
			return
		}

		//if r.MultipartForm != nil && r.MultipartForm.File != nil {

		// var files map[int]UploadResultFile
		// files = make(map[int]UploadResultFile)

		list := make([]UploadResultFile, 0, len(r.MultipartForm.File))

		for k := range r.MultipartForm.File {
			result, err := uploadFileHandler(r, k)
			if err != nil {
				log.Print("error:", err)
				renderError(w, result.Message, result.Code)
				return
			}
			list = append(list, result.File)
		}

		res := UploadResult{list}
		log.Print(res)
		encdata, err := json.Marshal(res)
		if err != nil {
			log.Print(err)
			renderError(w, "Server error", http.StatusInternalServerError)
			return
		}
		sendWebhook(whURL, encdata)

		w.Write([]byte("SUCCESS"))

	})
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errdata := UploadError{message, false}
	raw, err := json.Marshal(errdata)
	if err != nil {
		renderError(w, "INVALID_FILE", statusCode)
		return
	}
	w.Write([]byte(raw))
}

func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func sendWebhook(url string, data []byte) {
	fmt.Println("URL:>", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	// fmt.Println("response Headers:", resp.Header)
	// body, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println("response Body:", string(body))
}
