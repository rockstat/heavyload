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

type jsonHZ = map[string]interface{}

// ResponseStruct Type for error responces
type ResponseStruct struct {
	Message    string      `json:"message"`
	NotifyResp interface{} `json:"resp"`
	Payload    interface{} `json:"payload"`
}

// UploadedFile holds tmp name and contains meta data
type UploadedFile struct {
	OrigFn string `json:"orig_fn"`
	Fn     string `json:"fn"`
	Size   int64  `json:"size"`
}

// NotificationStruct using for webhook notification
type NotificationStruct struct {
	Success bool           `json:"success"`
	Files   []UploadedFile `json:"files"`
}

// UserTextWithStatus contain user and system info
type UserTextWithStatus struct {
	Text       string
	StatusCode int
}

// ResultWrapper response to uploader
type ResultWrapper struct {
	ResultCode int
	File       UploadedFile
}

const maxUploadSize = 32 << 18 // ~ 4mb
const uploadPath = "./upload"
const defaultAddr = ""
const defaultPort = 8080

const contentType = "Content-Type"
const typeJson = "application/json"

const defaultWebhookURL = "http://127.0.0.1:10001/wh/upload/notify"

const (
	resultOk    = 1
	invalidFile = 4001
	tooBig      = 4002
	noFiles     = 4003
	writeErr    = 5001
	encodeErr   = 5002
	notifyErr   = 5003
)

var resultText = map[int]UserTextWithStatus{
	resultOk:    UserTextWithStatus{"OK", http.StatusOK},
	invalidFile: UserTextWithStatus{"INVALID_FILE", http.StatusBadRequest},
	writeErr:    UserTextWithStatus{"CANT_WRITE_FILE", http.StatusInternalServerError},
	tooBig:      UserTextWithStatus{"FILE_SIZE_EXCEED", http.StatusBadRequest},
	noFiles:     UserTextWithStatus{"NO_FILES", http.StatusBadRequest},
	encodeErr:   UserTextWithStatus{"ENCODE_NOTIFICATION_ERROR", http.StatusInternalServerError},
	notifyErr:   UserTextWithStatus{"NOTIFICATION_ERROR", http.StatusInternalServerError},
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
		return ResultWrapper{invalidFile, UploadedFile{}}, err
	}

	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return ResultWrapper{invalidFile, UploadedFile{}}, err
	}

	fileName := randToken(12)
	newPath := filepath.Join(uploadPath, fileName)
	fmt.Printf("Form field key %s; Orig file name %s; File: %s\n", key, origFn, newPath)

	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		return ResultWrapper{writeErr, UploadedFile{}}, err
	}
	defer newFile.Close()
	if _, err := newFile.Write(fileBytes); err != nil {
		return ResultWrapper{writeErr, UploadedFile{}}, err
	}
	return ResultWrapper{resultOk, UploadedFile{origFn, fileName, header.Size}}, nil
}

func uploadHandler(whURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate file size

		http.MaxBytesReader(w, r.Body, maxUploadSize)
		// r.Body = ioutil.NopCloser(io.LimitReader(r.Body, maxUploadSize))
		log.Print("attached max bytes reader. parsing multipart")
		if err := r.ParseMultipartForm(32 << 19); err != nil {
			log.Print("err catched")
			jsonResponse(w, tooBig, nil, nil)
			log.Println(err)
			return
		}
		log.Print("err catched")
		if r.MultipartForm == nil || r.MultipartForm.File == nil {
			jsonResponse(w, noFiles, nil, nil)
			return
		}

		list := make([]UploadedFile, 0, len(r.MultipartForm.File))

		for k := range r.MultipartForm.File {
			result, err := uploadFileHandler(r, k)
			if err != nil {
				log.Print("error:", err)
				jsonResponse(w, result.ResultCode, nil, nil)
				return
			}
			list = append(list, result.File)
		}

		notifyPayl := NotificationStruct{true, list}
		encdata, err := json.Marshal(notifyPayl)
		if err != nil {
			log.Print(err)
			jsonResponse(w, encodeErr, nil, nil)
			return
		}

		body, err := sendWebhook(whURL, encdata)
		if err != nil {
			log.Print("webhook err", err)
			jsonResponse(w, notifyErr, nil, nil)
			return
		}

		var notifyResp jsonHZ
		err = json.Unmarshal(body, &notifyResp)
		if err != nil {
			log.Print("wh rest unmarshall", err)
			jsonResponse(w, notifyErr, nil, nil)
			return
		}
		jsonResponse(w, resultOk, &notifyResp, &notifyPayl)
	})
}

func jsonResponse(w http.ResponseWriter, resultCode int, notifyResp *jsonHZ, notifyPayl *NotificationStruct) {
	log.Print("called jsonResp", resultCode, notifyResp, notifyPayl)

	w.Header().Set(contentType, typeJson)
	w.WriteHeader(resultText[resultCode].StatusCode)

	resp := ResponseStruct{resultText[resultCode].Text, notifyResp, notifyPayl}
	raw, err := json.Marshal(resp)
	if err != nil {
		raw = []byte("\"json encode error\"")
	}
	w.Write([]byte(raw))
}

func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func sendWebhook(url string, data []byte) ([]byte, error) {
	fmt.Println("URL:>", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		fmt.Println("response Body:", string(body))
	}

	return body, nil
}

// https://stackoverflow.com/questions/28073395/limiting-file-size-in-formfile
// https://github.com/golang/go/issues/23165
// https://stackoverflow.com/questions/28282370/is-it-advisable-to-further-limit-the-size-of-forms-when-using-golang/28292505#28292505
// https://godoc.org/github.com/gin-gonic/gin
