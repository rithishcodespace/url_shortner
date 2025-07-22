package routes

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rithishcodespace/url_shortner/api/database"
	"github.com/rithishcodespace/url_shortner/api/models"
	"github.com/asaskevich/govalidator"
	"github.com/rithishcodespace/url_shortner/api/utils"
	"github.com/google/uuid"
)

func ShortenURL(c *gin.Context) {
	var body models.Request
	if err := c.ShouldBind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot Parse JSON"})
		return
	}

	r2 := database.CreateClient(1) // rate limiter DB
	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.ClientIP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.ClientIP(), os.Getenv("API_QUOTA"), 30*time.Minute).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.ClientIP()).Result()
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "Rate limit exceeded",
				"rate_limit_reset": limit / time.Minute,
			})
			return
		}
	}

	// URL validation
	if !govalidator.IsURL(body.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Prevent shortening own domain
	if !utils.IsDifferentDomain(body.URL) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "You can hack this system :)"})
		return
	}

	body.URL = utils.EnsureHttpPrefix(body.URL)

	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0) // URL storage DB
	defer r.Close()

	// Check if short ID already exists
	_, err = r.Get(database.Ctx, id).Result()
	if err != redis.Nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "URL Custom Short Already Exists"})
		return
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	// Store short URL
	err = r.Set(database.Ctx, id, body.URL, time.Duration(body.Expiry)*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Unable to connect to the Redis server",
		})
		return
	}

	// Decrease quota AFTER success
	r2.Decr(database.Ctx, c.ClientIP())

	// Construct response
	val, _ = r2.Get(database.Ctx, c.ClientIP()).Result()
	remaining, _ := strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.ClientIP()).Result()

	resp := models.Response{
		URL:             body.URL,
		CustomShort:     os.Getenv("DOMAIN") + "/" + id,
		Expiry:          body.Expiry,
		XRateRemaining:  remaining,
		XRateLimitReset: ttl / time.Minute,
	}

	c.JSON(http.StatusOK, resp)
}
