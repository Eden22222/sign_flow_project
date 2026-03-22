package main

import (
	"log"
	"net/http"
	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	_, err := db.InitPostgres()
	if err != nil {
		log.Fatalf("init postgres failed: %v", err)
	}
	log.Println("database connected successfully")

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		if err := db.CheckDatabaseHealth(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database unhealthy",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	router.RegisterRoutes(r)

	if err := r.Run(":8081"); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
