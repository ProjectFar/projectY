package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gormtest/internal/model"
	"net/http"
	"time"
)

var db *gorm.DB
var ctx = context.Background()
var client *redis.Client
var isAdmin bool
var loggedUser string

func Connect(c *gin.Context) {
	var err error
	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to Redis"})
		return
	}

	dbURI := "root:password@tcp(127.0.0.1:3306)/test_db"
	if db == nil {
		db, err = gorm.Open(mysql.Open(dbURI), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true}})
		if err != nil {
			panic("failed to connect database")
		}
	}

	if err := db.AutoMigrate(&model.Blog{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to migrate database schema"})
		return
	}

}

func Login(c *gin.Context) {
	user := c.DefaultQuery("user", "")
	password := c.DefaultQuery("password", "")

	var adminMap = map[string]string{
		"Faruk":  "password",
		"Admin1": "password1",
		"Admin2": "password2",
		"Admin3": "password3",
	}

	var userMap = map[string]string{
		"User1": "user1",
		"User2": "user2",
		"User3": "user3",
	}

	for User, Password := range adminMap {
		if User == user && Password == password {
			c.JSON(http.StatusOK, gin.H{"result": "Login as an admin successful"})
			isAdmin = true
			return
		}
	}
	for User, Password := range userMap {
		if User == user && Password == password {
			c.JSON(http.StatusOK, gin.H{"result": "Login as a user successful"})
			isAdmin = false
			loggedUser = user
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"result": "Login failed please use valid credentials"})
	isAdmin = false

	return
}

func GetBloge(c *gin.Context) {
	if isAdmin {
		cacheKey := "bloges"
		val, err := client.Get(ctx, cacheKey).Result()
		if err == nil {
			fmt.Println("Retrieved data from cache")
			var bloges []model.Blog
			err = json.Unmarshal([]byte(val), &bloges)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal cached data"})
				return
			}
			c.IndentedJSON(http.StatusOK, bloges)
			return
		}

		var bloges []model.Blog
		if err := db.Find(&bloges).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve blogs"})
			return
		}

		cachedData, err := json.Marshal(bloges)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal data for caching"})
			return
		}

		err = client.Set(ctx, cacheKey, cachedData, 10*time.Minute).Err()
		if err != nil {
			return
		}
		c.IndentedJSON(http.StatusOK, bloges)
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	}
}

func UpdateBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog model.Blog

	if err := db.Where("id = ?", id).First(&blog).Error; err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	if !isAdmin && blog.User != loggedUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&blog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&blog).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blog"})
		return
	}

	// Clear or update the cache
	client.Del(ctx, fmt.Sprintf("blog:%s", id))
	client.Del(ctx, "bloges")

	c.IndentedJSON(http.StatusOK, blog)
}

func GetBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog model.Blog

	cacheKey := fmt.Sprintf("blog:%s", id)
	val, err := client.Get(ctx, cacheKey).Result()
	if err == nil {
		fmt.Println("Retrieved data from cache")
		err = json.Unmarshal([]byte(val), &blog)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal cached data"})
			return
		}
		c.IndentedJSON(http.StatusOK, blog)
		return
	}

	if err := db.Where("id = ?", id).First(&blog).Error; err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	// Cache the retrieved blog
	cachedData, err := json.Marshal(blog)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal data for caching"})
		return
	}

	err = client.Set(ctx, cacheKey, cachedData, 10*time.Minute).Err()
	if err != nil {
		fmt.Println("Failed to cache blog:", err)
	}
	if !isAdmin && blog.User != loggedUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	} else {
		c.IndentedJSON(http.StatusOK, blog)
	}
}

func PostBlog(c *gin.Context) {
	if !isAdmin && loggedUser == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "please login"})
	} else {
		var newBlog model.Blog

		if err := c.ShouldBindJSON(&newBlog); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		db.Create(&newBlog)

		// Clear or update the cache
		client.Del(ctx, "bloges")

		cacheKey := fmt.Sprintf("blog:%s", newBlog.ID)

		cachedData, err := json.Marshal(newBlog)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal data for caching"})
			return
		}

		err = client.Set(ctx, cacheKey, cachedData, 10*time.Minute).Err()
		if err != nil {
			return
		}

		c.IndentedJSON(http.StatusOK, newBlog)
	}
}

func DeleteBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog model.Blog

	if !isAdmin && blog.User != loggedUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	} else if loggedUser != "" {
		if err := db.Where("id = ?", id).Delete(&blog).Error; err != nil {
			c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
			return
		}

		// Clear the cache
		client.Del(ctx, fmt.Sprintf("blog:%s", id))
		client.Del(ctx, "bloges")

		c.IndentedJSON(http.StatusOK, gin.H{"result": "Blog deleted"})
	}
	c.IndentedJSON(http.StatusUnauthorized, gin.H{"error": "please login"})
}
