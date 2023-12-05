package backend

import (
	"log"

	"github.com/gin-gonic/gin"
)

type Service struct {
	Id          int64
	Name        string `schema:"service-name"`
	Description string `schema:"service-description"`
	Host        string `schema:"service-host"`
	Port        uint16 `schema:"service-port"`
}

func SetUp(hostAndPort string) {
	log.Printf("Starting backend api service on %s", hostAndPort)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run(hostAndPort)
}

// func Create
