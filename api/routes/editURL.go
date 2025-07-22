package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rithishcodespace/url_shortner/api/database"
	"github.com/rithishcodespace/url_shortner/api/models"
)

func EditURL(c *gin.Context) {
	shortID := c.Param("shortID")
	var body models.Request

	// 1. Parse request body
	if err := c.ShouldBind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot Parse JSON",
		})
		return
	}

	// 2. Connect to Redis
	r := database.CreateClient(0)
	defer r.Close()

	// 3. Check if shortID exists
	val, err := r.Get(database.Ctx, shortID).Result()
	if err == redis.Nil || val == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "ShortID doesn't exist",
		})
		return
	}

	// 4. Update the existing URL and expiry
	err = r.Set(database.Ctx, shortID, body.URL, time.Duration(body.Expiry)*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Unable to update the shortened content",
		})
		return
	}

	// 5. Return success message
	c.JSON(http.StatusOK, gin.H{
		"message": "The content has been updated successfully!",
	})
}
