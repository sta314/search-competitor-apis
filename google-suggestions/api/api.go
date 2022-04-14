package api

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"net/http"
	"strconv"
	"strings"
)

var rateLimiter *rate.Limiter

func init() {
	if rate.Every(delayBetweenGoogleRequests) != 0 {
		rateLimiter = rate.NewLimiter(rate.Every(delayBetweenGoogleRequests), 1)
	} else {
		rateLimiter = nil
	}
}

// RespondSuggestions godoc
// @Summary      Main function of the API
// @Description  Takes the keyword and optional parameters then returns suggestions
// @Tags         Suggester
// @Accept       json
// @Produce      json
// @Param        keyword  query      string     true   "Keyword to search"
// @Param        hl       query      string     false  "Host language code to restrict results"
// @Param        gl       query      string     false  "Two-letter country code to restrict results"
// @Param        depth    query      int        false  "Suggestion depth"
// @Param        load     query      boolean    false  "Check database for cached results"
// @Param        save     query      boolean    false  "Save results to the database"
// @Param        proxy    query      string     false  "Proxy to use for request"
// @Success      200      {array}    packedSuggestion  "Suggestions returned"
// @Failure      400      object     apiError      "Invalid parameter"
// @Failure      404      object     apiError      "Not defined error"
// @Failure      500      object     apiError      "Api function failure"
// @Failure      503      object     apiError      "HTTP call failed"
// @Router       /get_suggestions [get]
func RespondSuggestions(c *gin.Context) {

	allParameters := turnQueryParametersToMap(c.Request.URL.Query())

	suggestions, err := GetSuggestions(allParameters)

	if err != nil {
		errorDetail := strings.Split(err.Error(), "---")
		if len(errorDetail) > 1 {
			statusCode, _ := strconv.Atoi(errorDetail[0])
			c.IndentedJSON(statusCode, apiError{StatusCode: statusCode, ErrorMessage: errorDetail[1]})
		} else {
			c.IndentedJSON(404, apiError{StatusCode: 404, ErrorMessage: errorDetail[0]})
		}
	} else {
		c.IndentedJSON(http.StatusOK, suggestions)
	}
}

//Main function for getting results which consists parameter checking and database i/o

func GetSuggestions(allParameters map[string]string) ([]packedSuggestion, error) {

	ctx := context.Background()

	//parse and check parameters
	var searchParams searchParameters
	err := checkAndExtractSearchParameters(allParameters, &searchParams)
	if err != nil {
		return nil, err
	}

	load := allParameters["load"]
	save := allParameters["save"]

	//try to load from database if desired
	if (load == "1" || load == "true") && isDBActive(ctx) {
		oldRecordDepth, err := getDepthOfCachedResults(searchParams, ctx)
		if err != nil {
			return nil, err
		}

		if oldRecordDepth >= searchParams.depth {
			results, err := loadResultsFromDB(searchParams, ctx)
			if err != nil {
				return nil, err
			}

			return results, nil
		}
	}

	//perform search
	suggestions, err := getLeveledSuggestionsUnpacked(searchParams)
	if err != nil {
		return nil, err
	}

	//save results to database if desired
	if (save == "1" || save == "true") && isDBActive(ctx) {
		err = saveResultsToDB(searchParams, suggestions, ctx)
		if err != nil {
			return nil, err
		}
	}

	return suggestions, nil
}

//Same as getLeveledSuggestions unpacks 2D slice as 1D slice

func getLeveledSuggestionsUnpacked(params searchParameters) ([]packedSuggestion, error) {
	allSuggestionPacks, err := getLeveledSuggestions(params)
	allSuggestions := make([]packedSuggestion, 0)
	for _, curPack := range allSuggestionPacks {
		for _, curSuggestion := range curPack {
			allSuggestions = append(allSuggestions, curSuggestion)
		}
	}
	return allSuggestions, err
}

//Uses depth parameter to decide how deep search will go to

func getLeveledSuggestions(params searchParameters) ([][]packedSuggestion, error) {

	//Getting root suggestion
	allSuggestions := make([][]packedSuggestion, 0)
	firstLevel, err := getStructuredSuggestions(params, nil, 1)
	curLevel := 1
	maxLevel := params.depth
	if err != nil {
		return nil, err
	}
	allSuggestions = append(allSuggestions, firstLevel)
	//

	curIndex := 0
	prevSuggestionAmount := 1
	suggestionCount := 0

	for maxLevel-curLevel >= 1 {
		chSuggestion := make(chan []packedSuggestion)
		chCount := make(chan int)
		var eg errgroup.Group
		for _, curMultiSuggestion := range allSuggestions[curIndex:] { // Iterates over previous level and calls helper func to get their suggestions
			curMultiSuggestionPass := curMultiSuggestion
			eg.Go(func() error {
				return getLeveledSuggestionSub(params, curMultiSuggestionPass, curLevel, chSuggestion, chCount)
			})
		}

		numOfSuggestions := len(allSuggestions) - curIndex

		for i := 0; i < numOfSuggestions; i++ {
			suggestionCount += <-chCount
		}

		for i := 0; i < suggestionCount; i++ {
			allSuggestions = append(allSuggestions, <-chSuggestion)
		}

		if err := eg.Wait(); err != nil {
			return allSuggestions, err
		}

		curIndex += prevSuggestionAmount
		prevSuggestionAmount = suggestionCount
		suggestionCount = 0

		curLevel += 1
	}
	return allSuggestions, nil
}

//Helper function for getLeveledSuggestions. Deals with 1 pack of suggestions.

func getLeveledSuggestionSub(params searchParameters, curMultiSuggestion []packedSuggestion, level int, chSuggestion chan []packedSuggestion, chCount chan int) error {
	suggestionCount := 0
	ch := make(chan []packedSuggestion)
	var eg errgroup.Group
	for _, curSuggestion := range curMultiSuggestion { // Iterates over each different suggestion obtained from a keyword and gets their separate suggestions
		curSuggestionPass := curSuggestion
		eg.Go(func() error {
			return getStructuredSuggestionsChanneled(
				searchParameters{keyword: curSuggestionPass.SuggestedKeyword, hl: params.hl, gl: params.gl},
				curSuggestionPass.BaseKeywords, level+1, ch)
		})
	}
	existingSuggestions := make([][]packedSuggestion, 0)
	for i := 0; i < len(curMultiSuggestion); i++ {
		curSuggestion := <-ch
		if curSuggestion != nil && len(curSuggestion) > 0 {
			suggestionCount += 1
			existingSuggestions = append(existingSuggestions, curSuggestion)
		}
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	chCount <- suggestionCount
	for _, curSuggestion := range existingSuggestions {
		chSuggestion <- curSuggestion
	}

	return nil
}

//Same getStructuredSuggestions but instead of returning results, writes them to channel

func getStructuredSuggestionsChanneled(params searchParameters, pastKeywords []string, level int, ch chan []packedSuggestion) error {
	suggestion, suggestErr := getStructuredSuggestions(params, pastKeywords, level)
	ch <- suggestion
	return suggestErr
}

//Returns suggestions as a structure and excludes base keyword

func getStructuredSuggestions(params searchParameters, pastKeywords []string, level int) ([]packedSuggestion, error) {
	allSuggestions, err := getSuggestionsBasic(params)

	if err != nil {
		return nil, err
	}

	allStructuredSuggestions := make([]packedSuggestion, 0)
	baseKeywords := make([]string, 0)
	if pastKeywords != nil {
		baseKeywords = append(baseKeywords, pastKeywords...)
	}

	for _, suggestion := range allSuggestions {
		if suggestion != params.keyword {
			allStructuredSuggestions = append(allStructuredSuggestions, packedSuggestion{suggestion, append(baseKeywords, params.keyword), level})
		}
	}

	return allStructuredSuggestions, nil
}

//Simplest function for getting suggestions, pass all parameters as map

func getSuggestionsBasic(params searchParameters) ([]string, error) {

	URL := generateURL(params)

	response, err := getResponseUsingURL(URL, params.proxy)
	if err != nil {
		return nil, err
	}

	body, err := readResponseBody(response)
	if err != nil {
		return nil, err
	}

	parsedResults := parseResponseBody(body)
	return parsedResults, nil
}
