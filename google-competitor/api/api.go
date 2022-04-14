package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"net/http"
	"strconv"
	"strings"
)

// RespondResults godoc
// @Summary      Main function of the API
// @Description  Takes the keyword and optional parameters then returns results
// @Tags         Getter
// @Accept       json
// @Produce      json
// @Param        keyword  query      string     true   "Keyword to check for"
// @Param        limit    query      int        true   "Amount of desired results"
// @Success      200      {array}    scoredURLs        "Results returned"
// @Failure      404      object     apiError            "Not defined error"
// @Failure      500      object     apiError          "Api function failure"
// @Router       /get_competitors [get]
func RespondResults(c *gin.Context) {

	allParameters := turnQueryParametersToMap(c.Request.URL.Query())
	limitInt, _ := strconv.Atoi(allParameters["limit"])
	results, err := returnCompetitors(allParameters["keyword"], limitInt, context.Background())

	if err != nil {
		errorDetail := strings.Split(err.Error(), "---")
		if len(errorDetail) > 1 {
			statusCode, _ := strconv.Atoi(errorDetail[0])
			c.IndentedJSON(statusCode, apiError{StatusCode: statusCode, ErrorMessage: errorDetail[1]})
		} else {
			c.IndentedJSON(404, apiError{StatusCode: 404, ErrorMessage: errorDetail[0]})
		}
	} else {
		c.IndentedJSON(http.StatusOK, results)
	}
}

//Main function for getting competitors

func returnCompetitors(keyword string, limit int, ctx context.Context) ([]scoredURLs, error) {

	var URLs []scoredURLs
	err := chConn.Exec(ctx, fmt.Sprintf("CREATE or REPLACE VIEW competitor_hit_words AS (SELECT DISTINCT keyword FROM %s WHERE cutToFirstSignificantSubdomain(url) LIKE '%%%s%%' AND lowerUTF8(keyword) NOT LIKE '%%%s%%' AND rank <= 10)", serpResultsTableName, keyword, keyword))
	if err != nil {
		return nil, &apiError{500, "Error occurred while executing clickhouse query"}
	}
	err = chConn.Exec(ctx, fmt.Sprintf("CREATE or REPLACE VIEW competitor_hit_words_scored AS SELECT keyword, cutToFirstSignificantSubdomain(url) AS url_domain, minus(101, arrayMin(groupArray(rank))) AS score FROM %s WHERE keyword IN competitor_hit_words AND url_domain NOT IN %s GROUP BY keyword, url_domain ORDER BY keyword, score DESC", serpResultsTableName, ignoredURLsTableName))
	if err != nil {
		return nil, &apiError{500, "Error occurred while executing clickhouse query"}
	}
	err = chConn.Select(ctx, &URLs, fmt.Sprintf("SELECT *, plus(1, rowNumberInAllBlocks()) AS place FROM (SELECT url_domain, sum(score) AS total_score FROM competitor_hit_words_scored GROUP BY url_domain ORDER BY total_score DESC LIMIT %d)", limit))
	if err != nil {
		return nil, &apiError{500, "Error occurred while executing clickhouse query"}
	}

	return URLs, nil
}
