package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultLanguage = "en"
const defaultLocation = "us"
const defaultLimit = 10
const baseURL = "https://www.google.com/search?filter=1&pws=0&start=0"

//Adjusts global rate limiter of the api, 0 to disable
const delayBetweenGoogleRequests = time.Second * 0

var validLanguages = []string{"af", "ak", "sq", "ws", "am", "ar", "hy", "az", "eu", "be", "bem", "bn", "bh", "xx-bork", "bs", "br", "bg", "bt", "km", "ca", "chr", "ny", "zh-cn", "zh-tw", "co", "hr", "cs", "da", "nl", "xx-elmer", "en", "eo", "et", "ee", "fo", "tl", "fi", "fr", "fy", "gaa", "gl", "ka", "de", "el", "kl", "gn", "gu", "xx-hacker", "ht", "ha", "haw", "iw", "hi", "hu", "is", "ig", "id", "ia", "ga", "it", "ja", "jw", "kn", "kk", "rw", "rn", "xx-klingon", "kg", "ko", "kri", "ku", "ckb", "ky", "lo", "la", "lv", "ln", "lt", "loz", "lg", "ach", "mk", "mg", "my", "ml", "mt", "mv", "mi", "mr", "mfe", "mo", "mn", "sr-me", "ne", "pcm", "nso", "no", "nn", "oc", "or", "om", "ps", "fa", "xx-pirate", "pl", "pt", "pt-br", "pt-pt", "pa", "qu", "ro", "rm", "nyn", "ru", "gd", "sr", "sh", "st", "tn", "crs", "sn", "sd", "si", "sk", "sl", "so", "es", "es-419", "su", "sw", "sv", "tg", "ta", "tt", "te", "th", "ti", "to", "lua", "tum", "tr", "tk", "tw", "ug", "uk", "ur", "uz", "vu", "vi", "cy", "wo", "xh", "yi", "yo", "zu"}
var validLocations = []string{"af", "al", "dz", "as", "ad", "ao", "ai", "aq", "ag", "ar", "am", "aw", "au", "at", "az", "bs", "bh", "bd", "bb", "by", "be", "bz", "bj", "bm", "bt", "bo", "ba", "bw", "bv", "br", "io", "bn", "bg", "bf", "bi", "kh", "cm", "ca", "cv", "ky", "cf", "td", "cl", "cn", "cx", "cc", "co", "km", "cg", "cd", "ck", "cr", "ci", "hr", "cu", "cy", "cz", "dk", "dj", "dm", "do", "ec", "eg", "sv", "gq", "er", "ee", "et", "fk", "fo", "fj", "fi", "fr", "gf", "pf", "tf", "ga", "gm", "ge", "de", "gh", "gi", "gr", "gl", "gd", "gp", "gu", "gt", "gn", "gw", "gy", "ht", "hm", "va", "hn", "hk", "hu", "is", "in", "id", "ir", "iq", "ie", "il", "it", "jm", "jp", "jo", "kz", "ke", "ki", "kp", "kr", "kw", "kg", "la", "lv", "lb", "ls", "lr", "ly", "li", "lt", "lu", "mo", "mk", "mg", "mw", "my", "mv", "ml", "mt", "mh", "mq", "mr", "mu", "yt", "mx", "fm", "md", "mc", "mn", "ms", "ma", "mz", "mm", "na", "nr", "np", "nl", "an", "nc", "nz", "ni", "ne", "ng", "nu", "nf", "mp", "no", "om", "pk", "pw", "ps", "pa", "pg", "py", "pe", "ph", "pn", "pl", "pt", "pr", "qa", "re", "ro", "ru", "rw", "sh", "kn", "lc", "pm", "vc", "ws", "sm", "st", "sa", "sn", "rs", "sc", "sl", "sg", "sk", "si", "sb", "so", "za", "gs", "es", "lk", "sd", "sr", "sj", "sz", "se", "ch", "sy", "tw", "tj", "tz", "th", "tl", "tg", "tk", "to", "tt", "tn", "tr", "tm", "tc", "tv", "ug", "ua", "ae", "uk", "gb", "us", "um", "uy", "uz", "vu", "ve", "vn", "vg", "vi", "wf", "eh", "ye", "zm", "zw"}
var validLimits = []int{10, 50, 100}

type packedResult struct {
	URL         string `json:"url" example:"example.com'"`
	Rank        int    `json:"rank" example:"1"`
	Title       string `json:"title" example:"Example Site"`
	Description string `json:"description" example:"Create online products and publish them."`
}

type searchParameters struct {
	keyword string
	hl      string
	gl      string
	limit   int
}

type apiError struct {
	StatusCode   int    `json:"status_code" example:"999"`
	ErrorMessage string `json:"message" example:"Error message..."`
}

func (e *apiError) Error() string {
	return strconv.Itoa(e.StatusCode) + "---" + e.ErrorMessage
}

//Returns query parameters as map

func turnQueryParametersToMap(parametersWithDuplicate url.Values) map[string]string {
	allParameters := make(map[string]string)
	for k, v := range parametersWithDuplicate {
		allParameters[k] = v[0]
	}
	return allParameters
}

//Checks parameters and parses then writes them to given struct

func checkAndExtractGoogleParameters(allParameters map[string]string, paramStruct *searchParameters) error {

	keyword := allParameters["keyword"]
	hl := allParameters["hl"]
	gl := allParameters["gl"]
	limit := allParameters["limit"]

	if len(keyword) == 0 {
		return &apiError{400, "search keyword must exist"}
	}

	if len(hl) != 0 && !sliceContainsString(hl, validLanguages) {
		return &apiError{400, "unsupported language(hl) : " + hl}
	}

	if len(gl) != 0 && !sliceContainsString(gl, validLocations) {
		return &apiError{400, "unsupported location(gl) : " + gl}
	}

	limitInt, err := strconv.Atoi(limit)
	if len(limit) != 0 && (err != nil || !sliceContainsInt(limitInt, validLimits)) {
		return &apiError{400, "unsupported limit : " + limit}
	}

	if hl == "" {
		hl = defaultLanguage
	}

	if gl == "" {
		gl = defaultLocation
	}

	if limitInt == 0 {
		limitInt = defaultLimit
	}

	*paramStruct = searchParameters{
		keyword: keyword,
		hl:      hl,
		gl:      gl,
		limit:   limitInt,
	}

	return nil
}

//Generates the request url

func generateURL(params searchParameters) string {
	newURL := baseURL + "&q=" + strings.ReplaceAll(params.keyword, " ", "+") + "&hl=" + params.hl + "&gl=" + params.gl + "&num=" + strconv.Itoa(params.limit)
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

//Functions for checking whether given slice contains given element

func sliceContainsString(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func sliceContainsInt(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
