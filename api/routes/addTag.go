package routes

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rithishcodespace/url_shortner/api/database"
)

type TagRequest struct {
	ShortID string `json:"shortID"`
	Tag     string `json:"tag"`
}

func AddTagic(c *gin.Context) {
	var TagRequest TagRequest
	if err := c.ShouldBindJSON(&TagRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H {
			"error": "Invalid Request Body",
		})
		return
	}
	shortID := TagRequest.ShortID
	tag := TagRequest.Tag

	r := database.CreateClient(0)
	defer r.Close()

	val,err := r.Get(database.Ctx, shortID).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":"Data not found for the given ShortID",
		})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		data = make(map[string]interface{})
		data["data"] = val
	}

	var tags []string
	if existingTags, ok := data["tags"].([]interface{}); ok {
		for _, t := range existingTags {
			if strTag, ok := t.(string); ok {
				tags = append(tags, strTag)
			}
		}
	}

	for _,existingTag := range tags {
		if existingTag == tag {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":"Tag Already Exists",
			})
			return
		}
	}

	tags = append(tags, tag)
	data["tag"] = tags

	updatedData, err := json.Marshal(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":"Failed to Marshal updated data",
		})
		return
	}

	err = r.Set(database.Ctx, shortID, updatedData, 0).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":"Failed to Update the Database",
		})
		return
	}

	c.JSON(http.StatusOK, data)

}