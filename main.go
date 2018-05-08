package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const maxUploadSize = 32 << 17 // 4m //5 * 1024 * 1024 // 5 MB
const uploadPath = "./upload"

// UploadError Type for error responces
type UploadError struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type UploadResult struct {
	OrigFn string `json:"orig_fn"`
	Fn     string `json:"fn"`
}

type ResultWrapper struct {
	Code    int
	Message string
	File    UploadResult
}

func main() {
	log.Printf("Server started on 0.0.0.0:18080, use /upload for uploading files. Max file size %d", maxUploadSize)
	http.HandleFunc("/upload", uploadHandler())
	log.Fatal(http.ListenAndServe("0.0.0.0:18080", nil))

}

func uploadFileHandler(r *http.Request, key string) (ResultWrapper, error) {

	file, header, err := r.FormFile(key)
	origFn := header.Filename
	if err != nil {
		return ResultWrapper{http.StatusBadRequest, "INVALID_FILE", UploadResult{}}, err
	}

	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return ResultWrapper{http.StatusBadRequest, "INVALID_FILE", UploadResult{}}, err
	}

	fileName := randToken(12)
	newPath := filepath.Join(uploadPath, fileName) //fileEndings[0]
	fmt.Printf("Form field key %s; Orig file name %s; File: %s\n", key, origFn, newPath)

	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		return ResultWrapper{http.StatusInternalServerError, "CANT_WRITE_FILE", UploadResult{}}, err
	}
	defer newFile.Close()
	if _, err := newFile.Write(fileBytes); err != nil {
		return ResultWrapper{http.StatusInternalServerError, "CANT_WRITE_FILE", UploadResult{}}, err
	}
	return ResultWrapper{http.StatusOK, "OK", UploadResult{origFn, fileName}}, nil

}

func sendWh(data []byte) {
	url := "http://127.0.0.1:10001/wh/upload/result"
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
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

func uploadHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate file size
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			renderError(w, "FILE_SIZE_EXCEED", http.StatusBadRequest)
			log.Println(err)
			return
		}

		//if r.MultipartForm != nil && r.MultipartForm.File != nil {

		var files map[int]UploadResult
		files = make(map[int]UploadResult)
		n := 0

		for k := range r.MultipartForm.File {
			// keys = append(keys, k)

			result, err := uploadFileHandler(r, k)
			if err != nil {
				log.Print("error:", err)
				renderError(w, result.Message, result.Code)
				return
			}
			files[n] = result.File
		}

		enc, err := json.Marshal(files)
		if err != nil {
			log.Print(err)
			renderError(w, "Servicer error", http.StatusInternalServerError)
			return
		}
		sendWh(enc)

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
