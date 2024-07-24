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
	"gormtest/internal"
	"net/http"
	"strconv"
	"time"
)

var db *gorm.DB
var ctx = context.Background()
var client *redis.Client

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

	if err := db.AutoMigrate(&internal.Blog{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to migrate database schema"})
		return
	}

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

	/*// Clear the cache for the updated blog
	cacheKey := fmt.Sprintf("blog:%s", id)
	err := client.Del(ctx, cacheKey).Err()
	if err != nil {
		fmt.Println("Failed to clear cache for updated blog:", err)
	}
	*/
	c.IndentedJSON(http.StatusOK, updatedBlog)
}
func GetBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog internal.Blog

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

	c.IndentedJSON(http.StatusOK, blog)
}

func PostBlog(c *gin.Context) {
	var newBlog internal.Blog

	if err := c.ShouldBindJSON(&newBlog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Create(&newBlog)

	cachedData, err := json.Marshal(newBlog)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal data for caching"})
		return
	}

	cacheKey := fmt.Sprintf("blog:%d", newBlog.ID)
	if ort := client.Set(ctx, cacheKey, cachedData, 10*time.Minute).Err; ort != nil {
		if ctx == nil {
			fmt.Println("ctx ist das problem")
		}
		if cachedData == nil {
			fmt.Println("chacheData ist das problem")
		} else {
			fmt.Println(ort)
		}
		if err := ctx.Err(); err != nil {
			fmt.Println("Failed to cache blog:", err)
		}
	}
	c.IndentedJSON(http.StatusOK, newBlog)
}

func DeleteBlogByID(c *gin.Context) {
	id := c.Param("id")
	var blog internal.Blog

	if err := db.Where("id = ?", id).Delete(&blog).Error; err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}
	idef, _ := strconv.Atoi(blog.ID)
	clearCache(ctx, idef)

	c.IndentedJSON(http.StatusOK, gin.H{"result": "Blog deleted"})
}

func clearCache(ctx context.Context, blogID int) error {
	key := fmt.Sprintf("blog:%d", blogID)
	err := client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to clear cache for blog %d: %w", blogID, err)
	}
	return nil
}
