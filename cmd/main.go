package main

import (
	"github.com/gin-gonic/gin"
	"gormtest/internal/handler"
)

func main() {
	var c = gin.Context{}
	handler.Connect(&c)
	router := gin.Default()
	router.POST("/login", handler.Login)
	router.GET("/bloge", handler.GetBloge)
	router.GET("/blog/:id", handler.GetBlogByID)
	router.POST("/blog", handler.PostBlog)
	router.DELETE("/blog/:id", handler.DeleteBlogByID)
	router.PUT("/blog/:id", handler.UpdateBlogByID)
	err := router.Run("localhost:6380")
	if err != nil {
		println("Error starting the server: %v", err)
		return
	}
}
