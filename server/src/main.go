package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    r.Static("/static", "./build/static")

    r.GET("/api/words/top", func(c *gin.Context) {
        words := getTop(10)
        c.JSON(200, words)
    })
    r.GET("/", func(c *gin.Context) {
        c.File("build/index.html")
    })


    go tweetsWorker()
    r.Run("0.0.0.0:8080")
}
