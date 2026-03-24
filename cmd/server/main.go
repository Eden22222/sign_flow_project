package main

import (
	"log"
	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/middleware/logging"
	"sign_flow_project/internal/router"
)

func init() {
	logging.Setup()

	_, err := db.PostgresSetup()
	if err != nil {
		log.Fatalf("init postgres failed: %v", err)
	}
	log.Println("database connected successfully")
}

func main() {
	r := router.InitRouter()

	if err := r.Run(":8081"); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
