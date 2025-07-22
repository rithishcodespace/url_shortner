package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rithishcodespace/url_shortner/api/database"
)

func getByShortID(c *gin.Context) {
  shortID := c.Params("shortID")

  r := database.CreateClient(0)
  defer r.Close()

  val,err := r.Get(database.Ctx, shortID).Result()
  if err != nil {
	c.JSON(http.StatusNotFound, gin.H{
		"error": "Data not found for given ShortID",
	})
	return
  }
  c.JSON(http.StatusOK, gin.H{"data":val})
}