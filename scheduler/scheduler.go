package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/afridhozega/dez-cron/db"
	"github.com/afridhozega/dez-cron/models"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var Cron *cron.Cron
var ActiveJobs = make(map[string]cron.EntryID)
var errLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags)

func Init() {
	Cron = cron.New()
	Cron.Start()
	ReloadAllJobs()
}

func ReloadAllJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.DB.Collection("jobs")
	cursor, err := collection.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		errLogger.Println("Error finding jobs:", err)
		return
	}
	defer cursor.Close(ctx)

	var jobs []models.JobConfig
	if err = cursor.All(ctx, &jobs); err != nil {
		errLogger.Println("Error decoding jobs:", err)
		return
	}

	for _, oldJob := range ActiveJobs {
		Cron.Remove(oldJob)
	}

	ActiveJobs = make(map[string]cron.EntryID)

	for _, j := range jobs {
		job := j // Capture loop variable
		
		// Set default timezone if missing
		tz := job.Timezone
		if tz == "" {
			tz = "Asia/Jakarta"
		}

		// Inject timezone
		cronStr := fmt.Sprintf("CRON_TZ=%s %s", tz, job.Schedule)

		entryID, err := Cron.AddFunc(cronStr, func() {
			ExecuteJob(job)
		})
		if err != nil {
			errLogger.Printf("Failed to schedule job %s: %v\n", job.ID.Hex(), err)
			continue
		}
		ActiveJobs[job.ID.Hex()] = entryID
	}
	log.Printf("Loaded & Scheduled %d active jobs\n", len(jobs))
}

func ExecuteJob(job models.JobConfig) {
	startTime := time.Now()

	client := &http.Client{Timeout: 30 * time.Second}
	var statusCode int = 0
	var errMsg string = ""
	var responseBody string = ""
	var isSuccess bool = false

	maxAttempts := 1 + job.RetryCount
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(job.Method, job.URL, bytes.NewBuffer([]byte(job.Body)))
		if err == nil {
			for k, v := range job.Headers {
				req.Header.Set(k, v)
			}
		}

		if err != nil {
			errMsg = err.Error()
			isSuccess = false
		} else {
			resp, reqErr := client.Do(req)
			if reqErr != nil {
				errMsg = reqErr.Error()
				isSuccess = false
			} else {
				statusCode = resp.StatusCode
				isSuccess = statusCode >= 200 && statusCode < 300
				
				limitReader := io.LimitReader(resp.Body, 10240)
				bodyBytes, _ := io.ReadAll(limitReader)
				resp.Body.Close()
				
				responseBody = string(bodyBytes)
				if len(bodyBytes) == 10240 {
					responseBody += "\n...[response truncated to 10KB]"
				}

				if !isSuccess {
					errMsg = "HTTP Status: " + resp.Status
				} else {
					errMsg = ""
				}
			}
		}

		if isSuccess {
			break
		} else if attempt < maxAttempts {
			time.Sleep(3 * time.Second) // Wait 3 seconds before retry
		}
	}

	duration := time.Since(startTime).Milliseconds()

	// Parse with Timezone
	tz := job.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	cronStr := fmt.Sprintf("CRON_TZ=%s %s", tz, job.Schedule)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, errP := parser.Parse(cronStr)
	
	nextExec := time.Time{}
	if errP == nil {
		nextExec = sched.Next(time.Now())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updateFields := bson.M{
		"last_execution":   startTime,
		"next_execution":   nextExec,
		"last_duration_ms": duration,
		"failed":           !isSuccess,
	}

	shouldDisable := false
	if isSuccess {
		updateFields["consecutive_failures"] = 0
	} else {
		newFailures := job.ConsecutiveFailures + 1
		updateFields["consecutive_failures"] = newFailures
		if newFailures >= job.DisabledAfter {
			updateFields["is_active"] = false
			shouldDisable = true
			log.Printf("Job %s disabled because it failed %d times consecutively.", job.Title, newFailures)
			go sendAlertWebhook(job, newFailures)
		}
	}

	update := bson.M{"$set": updateFields}
	_, updateErr := db.DB.Collection("jobs").UpdateOne(ctx, bson.M{"_id": job.ID}, update)
	if updateErr != nil {
		errLogger.Println("Failed to update job status in DB:", updateErr)
	}

	// Create Log
	logEntry := models.JobLog{
		ID:           primitive.NewObjectID(),
		JobID:        job.ID,
		StatusCode:   statusCode,
		DurationMs:   duration,
		IsSuccess:    isSuccess,
		ErrorMessage: errMsg,
		ResponseBody: responseBody,
		ExecutedAt:   startTime,
	}

	_, logErr := db.DB.Collection("job_logs").InsertOne(ctx, logEntry)
	if logErr != nil {
		errLogger.Println("Failed to write log to DB:", logErr)
	}
	
	if isSuccess {
		fmt.Printf("✅ Finished job %s (%s) in %d ms.\n", job.Title, job.ID.Hex(), duration)
	} else {
		errLogger.Printf("❌ Failed job %s (%s). HTTP: %d. Error: %s (Duration: %d ms)\n", job.Title, job.ID.Hex(), statusCode, errMsg, duration)
	}

	if shouldDisable {
		go ReloadAllJobs()
	}
}

func sendAlertWebhook(job models.JobConfig, failures int) {
	webhookUrl := job.AlertWebhookURL
	if webhookUrl == "" {
		return
	}
	payload := map[string]interface{}{
		"text": fmt.Sprintf("🚨 *Dez Cron Alert* 🚨\nJob *%s* has failed %d times consecutively and has been disabled.\nURL: %s", job.Title, failures, job.URL),
	}
	body, _ := json.Marshal(payload)
	
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		errLogger.Println("Failed to send alert webhook:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("Sent Alert Webhook.")
}
