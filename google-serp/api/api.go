package api

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"golang.org/x/net/html/charset"
	"golang.org/x/time/rate"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var rateLimiter *rate.Limiter

func init() {
	if rate.Every(delayBetweenGoogleRequests) != 0 {
		rateLimiter = rate.NewLimiter(rate.Every(delayBetweenGoogleRequests), 1)
	} else {
		rateLimiter = nil
	}
}

// RespondResults godoc
// @Summary      Main function of the API
// @Description  Takes the keyword and optional parameters then returns results
// @Tags         Getter
// @Accept       json
// @Produce      json
// @Param        keyword  query      string     true   "Keyword to search"
// @Param        hl       query      string     false  "Host language code to restrict results"
// @Param        gl       query      string     false  "Two-letter country code to restrict results"
// @Param        limit    query      int        false  "Amount of desired results"
// @Param        load     query      boolean    false  "Check database for cached results"
// @Param        save     query      boolean    false  "Save results to the database"
// @Param        proxy    query      string     false  "Proxy to use for request"
// @Success      200      {array}    packedResult  "Results returned"
// @Failure      400      object     apiError      "Invalid parameter"
// @Failure      404      object     apiError      "Not defined error"
// @Failure      500      object     apiError      "Api function failure"
// @Failure      503      object     apiError      "HTTP call failed"
// @Router       /get_serp [get]
func RespondResults(c *gin.Context) {

	allParameters := turnQueryParametersToMap(c.Request.URL.Query())

	results, err := GetResults(allParameters)

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

//Main function for getting results which consists parameter checking and database i/o

func GetResults(allParameters map[string]string) ([]packedResult, error) {

	ctx := context.Background()

	//parse and check parameters
	var googleParams searchParameters
	err := checkAndExtractGoogleParameters(allParameters, &googleParams)
	if err != nil {
		return nil, err
	}

	load := allParameters["load"]
	save := allParameters["save"]

	//try to load from database if desired
	if (load == "1" || load == "true") && isDBActive(ctx) {
		oldRecordLimit, err := getLimitOfCachedResults(googleParams, ctx)
		if err != nil {
			return nil, err
		}

		if oldRecordLimit >= googleParams.limit {
			results, err := loadResultsFromDB(googleParams, ctx)
			if err != nil {
				return nil, err
			}

			if len(results) > googleParams.limit {
				return results[:googleParams.limit], nil
			} else {
				return results, nil
			}
		}
	}

	//perform search
	results, err := getResultsBasic(googleParams, allParameters["proxy"])
	if err != nil {
		return nil, err
	}

	//save results to database if desired
	if (save == "1" || save == "true") && isDBActive(ctx) {
		err = saveResultsToDB(googleParams, results, ctx)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

//Simplest function for getting results for given parameters

func getResultsBasic(params searchParameters, proxy ...string) ([]packedResult, error) {
	URL := generateURL(params)

	response, err := getResponseFromGoogle(URL, proxy...)
	if err != nil {
		return nil, err
	}

	results, err := parseEngineResponse(response)
	if err != nil {
		return nil, err
	}

	return results, nil
}

//Parses Google response

func parseEngineResponse(res *http.Response) ([]packedResult, error) {

	defer res.Body.Close()

	//Creating goquery document for parsing
	contentType := res.Header.Get("Content-Type")
	suitableReader, err := charset.NewReader(res.Body, contentType)
	if err != nil {
		//debug purposes
		fmt.Println(err.Error())
		return nil, &apiError{500, "problem with parsing response"}
	}
	doc, err := goquery.NewDocumentFromReader(suitableReader)
	if err != nil {
		//debug purposes
		fmt.Println(err.Error())
		return nil, &apiError{500, "problem with parsing response body"}
	}
	//

	results := make([]packedResult, 0)

	parseDocumentWriteResults(doc, &results)

	return results, err
}

//Parses document using goquery

func parseDocumentWriteResults(doc *goquery.Document, results *[]packedResult) {
	sel := doc.Find("h3")

	rank := 1
	for i := range sel.Nodes {

		item := sel.Eq(i)
		div := item.Parent()
		divParent := div.Parent()
		descSel := divParent.Next()

		URL, valid := div.Attr("href")
		dropAfter := strings.Index(URL, "&sa=U")
		if !valid || dropAfter == -1 {
			continue
		}
		///Extract link from : url?q=https://example.com/&sa=U&ved=...
		URL, _ = url.PathUnescape(URL[7:dropAfter])

		title := item.Text()
		desc := descSel.Text()
		result := packedResult{
			URL:         URL,
			Rank:        rank,
			Title:       title,
			Description: desc,
		}
		*results = append(*results, result)

		rank++
	}
}

//Uses given url to get response

func getResponseFromGoogle(link string, proxy ...string) (*http.Response, error) {

	myClient := getHTTPClient(time.Minute*10, proxy...)
	req, _ := http.NewRequest(http.MethodGet, link, nil)

	//rate limiter
	if rateLimiter != nil {
		ctx := context.Background()
		if err := rateLimiter.Wait(ctx); err != nil {
			return nil, &apiError{500, "Rate limiter failed"}
		}
	}
	//

	res, err := myClient.Do(req)

	if err != nil {
		fmt.Println(err.Error())
		return nil, &apiError{503, "HTTP call failed"}
	}

	if res.StatusCode != 200 {
		return nil, &apiError{res.StatusCode, "response from Google is not OK, possible ban"}
	}

	return res, nil
}
