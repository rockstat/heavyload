package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
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
const typeJSON = "application/json"

const defaultWebhookURL = ""

// FileInfo for describe uploaded files
type FileInfo struct {
	Param    string `json:"param"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	TempName string `json:"tempName"`
}

func sendWebhook(url string, data []byte) ([]byte, error) {
	log.Print("URL:>", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("RESPONSE: status:%s\nheaders:%s\nbody:%s\n", resp.Status, resp.Header, string(body))
	}

	return body, nil
}

func getEnv(name string, def string) string {
	whEnv := os.Getenv(name)
	if whEnv != "" {
		return whEnv
	}
	return def
}

func main() {
	// params
	bindAddr := flag.String("addr", defaultAddr, "host addr to bund")
	bindPort := flag.Int("port", defaultPort, "listen port")
	whParam := flag.String("wh", defaultWebhookURL, "webhook to")
	flag.Parse()
	// envs
	webhookURL := getEnv("WEBHOOK", *whParam)
	if webhookURL == "" {
		panic("Webhook template not configured")
	}
	webhookTemplate, err := template.New("test").Parse(webhookURL)
	if err != nil {
		panic("Webhook template cant compiled")
	}

	listen := fmt.Sprintf("%s:%d", *bindAddr, *bindPort)
	log.Printf("Listening: %s", listen)
	log.Printf("Webhook: %s", webhookURL)

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 22 // 32 MiB
	r.POST("/upload/:service/:name", func(c *gin.Context) {

		query := make(map[string]string)
		// Handling query params
		q := c.Request.URL.Query()
		for key := range q {
			query[key] = q.Get(key)
		}
		form, err := c.MultipartForm()
		if err != nil {
			log.Printf("[ERROR] MultipartForm %v", err)
			return
		}
		files := []FileInfo{}
		// Handling query params
		for propName, propList := range form.File {
			for _, file := range propList {
				u4, err := uuid.NewV4()
				if err != nil {
					log.Printf("[ERROR] uuid.NewV4 %v", err)
					continue
				}
				tempName, _ := u4.MarshalText()
				if err != nil {
					log.Printf("[ERROR] u4.MarshalText %v", err)
					continue
				}
				dest := filepath.Join(uploadPath, string(tempName))
				err = c.SaveUploadedFile(file, dest)
				if err != nil {
					log.Printf("[ERROR] SaveUploadedFile %v", err)
					continue
				}
				files = append(files, FileInfo{propName, file.Filename, file.Size, string(tempName)})
			}
		}

		data := gin.H{
			"service": c.Param("service"),
			"name":    c.Param("name"),
			"query":   query,
			"files":   files,
		}
		raw, err := json.Marshal(data)
		if err != nil {
			log.Printf("[ERROR] json.Marshal %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprint(err)})
			return
		}

		var buf bytes.Buffer
		err = webhookTemplate.Execute(&buf, data)
		if err != nil {
			log.Printf("[ERROR]webhookTemplate.Execute %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprint(err)})
		}

		url := buf.String()
		_, err = sendWebhook(url, raw)
		if err != nil {
			log.Printf("[ERROR] sendWebhook %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprint(err)})
			return
		}

		c.JSON(http.StatusOK, data)
	})
	r.GET("/upload", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"response": "kuku",
		})
	})

	r.Run(listen)
}
