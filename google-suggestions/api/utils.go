package api

import (
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const defaultLanguage = "en"
const defaultLocation = "us"
const defaultDepth = 1
const baseURL = "https://google.com/complete/search?output=toolbar"

//Adjusts global rate limiter of the api, 0 to disable
const delayBetweenGoogleRequests = time.Second * 0

var validLanguages = []string{"af", "ak", "sq", "ws", "am", "ar", "hy", "az", "eu", "be", "bem", "bn", "bh", "xx-bork", "bs", "br", "bg", "bt", "km", "ca", "chr", "ny", "zh-cn", "zh-tw", "co", "hr", "cs", "da", "nl", "xx-elmer", "en", "eo", "et", "ee", "fo", "tl", "fi", "fr", "fy", "gaa", "gl", "ka", "de", "el", "kl", "gn", "gu", "xx-hacker", "ht", "ha", "haw", "iw", "hi", "hu", "is", "ig", "id", "ia", "ga", "it", "ja", "jw", "kn", "kk", "rw", "rn", "xx-klingon", "kg", "ko", "kri", "ku", "ckb", "ky", "lo", "la", "lv", "ln", "lt", "loz", "lg", "ach", "mk", "mg", "my", "ml", "mt", "mv", "mi", "mr", "mfe", "mo", "mn", "sr-me", "ne", "pcm", "nso", "no", "nn", "oc", "or", "om", "ps", "fa", "xx-pirate", "pl", "pt", "pt-br", "pt-pt", "pa", "qu", "ro", "rm", "nyn", "ru", "gd", "sr", "sh", "st", "tn", "crs", "sn", "sd", "si", "sk", "sl", "so", "es", "es-419", "su", "sw", "sv", "tg", "ta", "tt", "te", "th", "ti", "to", "lua", "tum", "tr", "tk", "tw", "ug", "uk", "ur", "uz", "vu", "vi", "cy", "wo", "xh", "yi", "yo", "zu"}
var validLocations = []string{"af", "al", "dz", "as", "ad", "ao", "ai", "aq", "ag", "ar", "am", "aw", "au", "at", "az", "bs", "bh", "bd", "bb", "by", "be", "bz", "bj", "bm", "bt", "bo", "ba", "bw", "bv", "br", "io", "bn", "bg", "bf", "bi", "kh", "cm", "ca", "cv", "ky", "cf", "td", "cl", "cn", "cx", "cc", "co", "km", "cg", "cd", "ck", "cr", "ci", "hr", "cu", "cy", "cz", "dk", "dj", "dm", "do", "ec", "eg", "sv", "gq", "er", "ee", "et", "fk", "fo", "fj", "fi", "fr", "gf", "pf", "tf", "ga", "gm", "ge", "de", "gh", "gi", "gr", "gl", "gd", "gp", "gu", "gt", "gn", "gw", "gy", "ht", "hm", "va", "hn", "hk", "hu", "is", "in", "id", "ir", "iq", "ie", "il", "it", "jm", "jp", "jo", "kz", "ke", "ki", "kp", "kr", "kw", "kg", "la", "lv", "lb", "ls", "lr", "ly", "li", "lt", "lu", "mo", "mk", "mg", "mw", "my", "mv", "ml", "mt", "mh", "mq", "mr", "mu", "yt", "mx", "fm", "md", "mc", "mn", "ms", "ma", "mz", "mm", "na", "nr", "np", "nl", "an", "nc", "nz", "ni", "ne", "ng", "nu", "nf", "mp", "no", "om", "pk", "pw", "ps", "pa", "pg", "py", "pe", "ph", "pn", "pl", "pt", "pr", "qa", "re", "ro", "ru", "rw", "sh", "kn", "lc", "pm", "vc", "ws", "sm", "st", "sa", "sn", "rs", "sc", "sl", "sg", "sk", "si", "sb", "so", "za", "gs", "es", "lk", "sd", "sr", "sj", "sz", "se", "ch", "sy", "tw", "tj", "tz", "th", "tl", "tg", "tk", "to", "tt", "tn", "tr", "tm", "tc", "tv", "ug", "ua", "ae", "uk", "gb", "us", "um", "uy", "uz", "vu", "ve", "vn", "vg", "vi", "wf", "eh", "ye", "zm", "zw"}

type packedSuggestion struct {
	SuggestedKeyword string   `json:"suggested_keyword" example:"example pricing"`
	BaseKeywords     []string `json:"base_keywords" example:"example"`
	Level            int      `json:"level" example:"1"`
}

type searchParameters struct {
	keyword string
	hl      string
	gl      string
	depth   int
	proxy   string
}

type apiError struct {
	StatusCode   int    `json:"status_code" example:"999"`
	ErrorMessage string `json:"message" example:"Error message..."`
}

func (e *apiError) Error() string {
	return strconv.Itoa(e.StatusCode) + "---" + e.ErrorMessage
}

//Generates the request url

func generateURL(params searchParameters) string {
	newURL := baseURL + "&q=" + url.QueryEscape(params.keyword) + "&hl=" + params.hl + "&gl=" + params.gl
	return newURL
}

//Returns HTTP client for executing request

func getHTTPClient(timeout time.Duration, proxy ...string) *http.Client {
	var newClient *http.Client
	if len(proxy) > 0 && len(proxy[0]) > 0 {
		proxyUrl, _ := url.Parse(proxy[0])
		newClient = &http.Client{Timeout: timeout, Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	} else {
		newClient = &http.Client{Timeout: timeout}
	}

	return newClient
}

//Makes a call to get response using given url

func getResponseUsingURL(link string, proxy ...string) (*http.Response, error) {

	myClient := getHTTPClient(time.Second*10, proxy...)
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
		return nil, &apiError{503, "HTTP call failed"}
	}

	if res.StatusCode != 200 {
		return nil, &apiError{res.StatusCode, "response from Google is not OK, possible ban"}
	}

	return res, nil
}

//Reads response body

func readResponseBody(res *http.Response) (string, error) {
	contentType := res.Header.Get("Content-Type")
	suitableReader, err := charset.NewReader(res.Body, contentType)
	if err != nil {
		return "", &apiError{res.StatusCode, "error occurred while reading response body"}
	}
	body, err := ioutil.ReadAll(suitableReader)
	if err != nil {
		return "", &apiError{res.StatusCode, "error occurred while reading response body"}
	}
	escapedBody := html.UnescapeString(string(body))
	return escapedBody, nil
}

//Parses fetched xml data and returns suggested words as a slice

func parseResponseBody(xmlBulk string) []string {
	rex := regexp.MustCompile(`="([^"]+)"/>`)
	out := rex.FindAllStringSubmatch(xmlBulk, -1)

	res := make([]string, 0)
	for _, i := range out {
		res = append(res, i[1])
	}

	return res
}

//Checks parameters and parses then writes them to given struct

func checkAndExtractSearchParameters(allParameters map[string]string, paramStruct *searchParameters) error {

	keyword := allParameters["keyword"]
	hl := allParameters["hl"]
	gl := allParameters["gl"]
	depth := allParameters["depth"]
	proxy := allParameters["proxy"]

	if len(keyword) == 0 {
		return &apiError{400, "search keyword must exist"}
	}

	if len(hl) != 0 && !sliceContainsString(hl, validLanguages) {
		return &apiError{400, "unsupported language(hl) : " + hl}
	}

	if len(gl) != 0 && !sliceContainsString(gl, validLocations) {
		return &apiError{400, "unsupported location(gl) : " + gl}
	}

	depthInt, err := strconv.Atoi(depth)
	if len(depth) != 0 && (err != nil || depthInt < 1) {
		return &apiError{400, "unsupported depth (must be positive int) : " + depth}
	}

	if hl == "" {
		hl = defaultLanguage
	}

	if gl == "" {
		gl = defaultLocation
	}

	if depthInt == 0 {
		depthInt = defaultDepth
	}

	*paramStruct = searchParameters{
		keyword: keyword,
		hl:      hl,
		gl:      gl,
		depth:   depthInt,
		proxy:   proxy,
	}

	return nil
}

//Returns query parameters as map

func turnQueryParametersToMap(parametersWithDuplicate url.Values) map[string]string {
	allParameters := make(map[string]string)
	for k, v := range parametersWithDuplicate {
		allParameters[k] = v[0]
	}

	return allParameters
}

//Function to check whether given slice contains given element

func sliceContainsString(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
