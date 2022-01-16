package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
    r.Static("/public", "./public")

	r.GET("/api/words/top", func(c *gin.Context) {
        words := getTop(10)
		c.JSON(200, words)
	})
    r.GET("/", func(c *gin.Context) {
        c.File("public/index.html")
    })

    go tweetsWorker()
    r.Run("0.0.0.0:8080")
}
