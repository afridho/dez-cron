package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JobConfig struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title               string             `bson:"title" json:"title"`
	URL                 string             `bson:"url" json:"url"`
	Method              string             `bson:"method" json:"method"`
	Schedule            string             `bson:"schedule" json:"schedule"`
	Timezone            string             `bson:"timezone" json:"timezone"`
	IsActive            bool               `bson:"is_active" json:"is_active"`
	Headers             map[string]string  `bson:"headers,omitempty" json:"headers,omitempty"`
	Body                string             `bson:"body,omitempty" json:"body,omitempty"`
	RetryCount          int                `bson:"retry_count" json:"retry_count"`
	DisabledAfter       int                `bson:"disabled_after" json:"disabled_after"`
	AlertWebhookURL     string             `bson:"alert_webhook_url,omitempty" json:"alert_webhook_url,omitempty"`
	ConsecutiveFailures int                `bson:"consecutive_failures" json:"consecutive_failures"`
	CreatedAt           time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time          `bson:"updated_at" json:"updated_at"`
	LastExecution       time.Time          `bson:"last_execution,omitempty" json:"last_execution,omitempty"`
	NextExecution       time.Time          `bson:"next_execution,omitempty" json:"next_execution,omitempty"`
	Failed              bool               `bson:"failed" json:"failed"`
}

type JobLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	JobID        primitive.ObjectID `bson:"job_id" json:"job_id"`
	StatusCode   int                `bson:"status_code" json:"status_code"`
	DurationMs   int64              `bson:"duration_ms" json:"duration_ms"`
	IsSuccess    bool               `bson:"is_success" json:"is_success"`
	ErrorMessage string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
	ResponseBody string             `bson:"response_body,omitempty" json:"response_body,omitempty"`
	ExecutedAt   time.Time          `bson:"executed_at" json:"executed_at"`
}

type SysConfig struct {
	ID     string   `bson:"_id" json:"id"`       // typically "global"
	Tokens []string `bson:"tokens" json:"tokens"` // Multiple API access tokens
}
