package main

import (
	"github.com/gin-gonic/gin"
	"gormtest/internal/handler"
)

func main() {
	handler.Connect()
	router := gin.Default()
	router.GET("/bloge", handler.GetBloge)
	router.GET("/blog/:id", handler.GetBlogByID)
	router.POST("/blog", handler.PostBlog)
	router.DELETE("/blog/:id", handler.DeleteBlogByID)
	router.PUT("/blog/:id", handler.UpdateBlogByID)
	err := router.Run("localhost:8082")
	if err != nil {
		println("Error starting the server: %v", err)
		return
	}
}
