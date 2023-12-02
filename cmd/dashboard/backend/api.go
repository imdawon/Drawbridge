package backend

import (
	"log"

	"github.com/gin-gonic/gin"
)

func SetUpAPI(hostAndPort string) {
	log.Printf("Starting backend api service on %s", hostAndPort)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run(hostAndPort)
}
