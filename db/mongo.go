package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var DB *mongo.Database

func ConnectDB() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/dez-cron"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(100).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(5 * time.Minute)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	log.Println("Connected to MongoDB successfully")
	Client = client
	
	// Default db name can be parsed or hardcoded for now
	DB = client.Database("dez_cron")

	initIndexes(context.Background())
}

func initIndexes(ctx context.Context) {
	// TTL index for 14 days (1209600 seconds) on executed_at field in job_logs collection
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"executed_at": 1},
		Options: options.Index().SetExpireAfterSeconds(14 * 24 * 60 * 60),
	}
	_, err := DB.Collection("job_logs").Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Println("Warning: Failed to create TTL index for job_logs:", err)
	} else {
		log.Println("TTL index for job_logs (2 weeks expiration) ensured.")
	}
}
