package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gormtest/internal"
	"net/http"
)

var db *gorm.DB

func Connect() {
	var err error = nil
	dbURI := "root:password@tcp(127.0.0.1:3306)/test_db"

	db, err = gorm.Open(mysql.Open(dbURI), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true}})

	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&internal.Blog{})
}

func GetBloge(c *gin.Context) {
	var bloges []internal.Blog

	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection is nil"})
		return
	}

	if err := db.Find(&bloges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve bloges"})
		return
	}
	c.IndentedJSON(http.StatusOK, bloges)

}

func UpdateBlogByID(c *gin.Context) {
	var updatedBlog internal.Blog
	id := c.Param("id")

	if err := db.Where("id = ?", id).First(&updatedBlog).Error; err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	if err := c.ShouldBindJSON(&updatedBlog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Save(&updatedBlog)
	c.IndentedJSON(http.StatusOK, updatedBlog)
}
func GetBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog internal.Blog
	if err := db.Where("id = ?", id).First(&blog).Error; err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, blog)
}

func PostBlog(c *gin.Context) {
	var newBlog internal.Blog

	if err := c.ShouldBindJSON(&newBlog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&newBlog)

	c.IndentedJSON(http.StatusCreated, newBlog)
}
func DeleteBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog internal.Blog

	if err := db.Where("id = ?", id).Delete(&blog).Error; err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"result": "Blog deleted"})
}
