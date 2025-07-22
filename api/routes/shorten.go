// we are implemnenting rate limit, we redis key -> usersIP value -> number of requests(api_quota)
// during each request we are checking whether the user exists in the database, if not create key value pair with 30 minutes duration, else if present decrease the quota values(no of request per user), if key is not expired and available quota is <= 0 means gives error



package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rithishcodespace/url_shortner/api/database"
	"github.com/rithishcodespace/url_shortner/api/models"
	"github.com/asaskevich/govalidator"
	"github.com/rithishcodespace/url_shortner/api/utils"
	"github.com/google/uuid"
)

func ShortenURL(c *gin.Context){
	var body models.Request
	if err := c.ShouldBind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":"Cannot Parse JSON"})
		return;
	}

	r2 := database.CreateClient(1) 
	defer r2.Close()

	val, err := r2.Get(database.Ctx, c.ClientIP()).Result()
	if err == redis.Nil{
       _ = r2.Set(database.Ctx, c.ClientIP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ = r2.Get(database.Ctx, c.ClientIP()).Result()
		valInt, _ := strconv.Atoi(val)

		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.ClientIP()).Result()
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":"rate limit exceeded",
				"rate_limit_reset":limit/time.Nanosecond/time.Minute,
			})
			return
		}
	}

	if !govalidator.IsURL(body.URL) { // validator package
		c.JSON(http.StatusBadRequest, gin.H{"error":"Invalid URL"})
		return
	}

	if !utils.IsDifferentDomain(body.URL){
		c.JSON(http.StatusServiceUnavailable, gin.H{"error":"You can hack this system :)"})
		return
	}

	body.URL = utils.EnsureHttpPrefix(body.URL)

	var id string 

	if  body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.Custom
	}

	r := database.CreateClient(0)
    defer r.Close()

	if val != "" {
	c.JSON(http.StatusForbidden, gin.H{
		"error": "URL Custom Short Already Exists",
	})
	return
}


	if body.Expiry === 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error" : "Unable to connect to the redis server",
		})
		return;
	}

	resp := models.Response {
		Expiry : body.Expiry,
		XRateLimitReset : 30,
		XRateRemaining : 10,
		URL : body.URL,
		CustomShort: "",
	}

	r2.Decr(database.Ctx, c.ClientIP())

	val, _ = r2.Get(database.Ctx, c.ClientIP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ = r2.TTL(database.Ctx, c.ClientIP()).Result
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	c.JSON(http.StatusOK, resp)
}