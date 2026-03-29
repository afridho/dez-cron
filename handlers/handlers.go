package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/afridhozega/dez-cron/db"
	"github.com/afridhozega/dez-cron/models"
	"github.com/afridhozega/dez-cron/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RegisterRoutes(r *gin.Engine) {
	// Protected API Routes
	api := r.Group("/api/jobs")
	api.Use(AuthMiddleware()) // require bearer token!

	api.GET("", GetJobs)
	api.GET("/:id", GetJob)
	api.POST("/:id/run", RunJob)
	api.POST("", CreateJob)
	api.PUT("/:id", UpdateJob)
	api.DELETE("/:id", DeleteJob)
	
	api.GET("/logs", GetLogs)
	api.GET("/logs/:job_id", GetJobLogs)
}

func GetJobs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("jobs").Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}
	defer cursor.Close(ctx)

	var jobs []models.JobConfig
	if err = cursor.All(ctx, &jobs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode jobs"})
		return
	}
	if jobs == nil {
		jobs = make([]models.JobConfig, 0)
	}

	c.JSON(http.StatusOK, jobs)
}

func CreateJob(c *gin.Context) {
	var job models.JobConfig
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if job.Timezone == "" {
		job.Timezone = "Asia/Jakarta"
	}

	// Validate cron
	cronStr := fmt.Sprintf("CRON_TZ=%s %s", job.Timezone, job.Schedule)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(cronStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cron expression or timezone: " + err.Error()})
		return
	}

	job.ID = primitive.NewObjectID()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	// New feature defaults
	if job.RetryCount == 0 {
		job.RetryCount = 5
	}
	if job.DisabledAfter == 0 {
		job.DisabledAfter = 20
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = db.DB.Collection("jobs").InsertOne(ctx, job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	scheduler.ReloadAllJobs() // reload scheduling

	c.JSON(http.StatusCreated, job)
}

func GetJob(c *gin.Context) {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var job models.JobConfig
	err = db.DB.Collection("jobs").FindOne(ctx, bson.M{"_id": id}).Decode(&job)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func RunJob(c *gin.Context) {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var job models.JobConfig
	if err := db.DB.Collection("jobs").FindOne(ctx, bson.M{"_id": id}).Decode(&job); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	go scheduler.ExecuteJob(job) // executing in background

	c.JSON(http.StatusOK, gin.H{"message": "Job execution triggered. Check logs endpoint for results."})
}

func UpdateJob(c *gin.Context) {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check timezone 
	var tz string
	if val, ok := req["timezone"].(string); ok {
		tz = val
	} else {
		// fetch current timezone
		var oldJob models.JobConfig
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		db.DB.Collection("jobs").FindOne(ctx2, bson.M{"_id": id}).Decode(&oldJob)
		cancel2()
		tz = oldJob.Timezone
	}

	if schedule, ok := req["schedule"].(string); ok {
		cronStr := fmt.Sprintf("CRON_TZ=%s %s", tz, schedule)
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		_, err := parser.Parse(cronStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cron expression: " + err.Error()})
			return
		}
	}

	req["updated_at"] = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{"$set": req}
	_, err = db.DB.Collection("jobs").UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}

	scheduler.ReloadAllJobs()

	var updatedJob models.JobConfig
	FindErr := db.DB.Collection("jobs").FindOne(ctx, bson.M{"_id": id}).Decode(&updatedJob)
	if FindErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated job after edit"})
		return
	}

	c.JSON(http.StatusOK, updatedJob)
}

func DeleteJob(c *gin.Context) {
	idParam := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = db.DB.Collection("jobs").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete job"})
		return
	}

	// Delete associated logs
	_, logErr := db.DB.Collection("job_logs").DeleteMany(ctx, bson.M{"job_id": id})
	if logErr != nil {
		fmt.Printf("Warning: Failed to delete job logs for %s: %v\n", id.Hex(), logErr)
	}

	scheduler.ReloadAllJobs()

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

func GetLogs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("job_logs").Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
		return
	}
	defer cursor.Close(ctx)

	var logs []models.JobLog
	if err = cursor.All(ctx, &logs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode logs"})
		return
	}
	
	if logs == nil {
		logs = make([]models.JobLog, 0)
	}

	c.JSON(http.StatusOK, logs)
}

func GetJobLogs(c *gin.Context) {
	idParam := c.Param("job_id")
	jobID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Job ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("job_logs").Find(ctx, bson.M{"job_id": jobID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
		return
	}
	defer cursor.Close(ctx)

	var logs []models.JobLog
	if err = cursor.All(ctx, &logs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}
