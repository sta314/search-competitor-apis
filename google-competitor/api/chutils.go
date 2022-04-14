package api

import (
	"bufio"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"golang.org/x/net/context"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"
)

const serpResultsTableName = "google_serp_results"
const ignoredURLsTableName = "competitor_ignored_URLs"
const ignoredURLsFileName = "ignoredURLs.txt"

var chConn driver.Conn = nil

type scoredURLs struct {
	UrlDomain string `json:"url_domain" ch:"url_domain"`
	Score     int64  `json:"score" ch:"total_score"`
	Place     uint64 `json:"place" ch:"place"`
}

type apiError struct {
	StatusCode   int    `json:"status_code" example:"999"`
	ErrorMessage string `json:"message" example:"Error message..."`
}

func (e *apiError) Error() string {
	return strconv.Itoa(e.StatusCode) + "---" + e.ErrorMessage
}

//Initially, connect to DB and create & fill ignored URL table using URLs in provided file

func init() {
	ctx := context.Background()

	err := establishClickhouseConnection()
	if err != nil {
		log.Println(err)
		return
	}

	err = createIgnoredURLsTable(true, ctx)
	if err != nil {
		log.Println(err)
		return
	}

	err = fileReadIgnoredURLsThenWriteToTable(ignoredURLsFileName, ctx)
	if err != nil {
		log.Println(err)
		return
	}
}

func establishClickhouseConnection() error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		//Debug:           true,
		DialTimeout:     time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})

	if err == nil {
		chConn = conn
	} else {
		return &apiError{500, "Error occurred while connecting to clickhouse"}
	}

	return err
}

func createIgnoredURLsTable(dropIfExists bool, ctx context.Context) error {

	if dropIfExists {
		if err := chConn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", ignoredURLsTableName)); err != nil {
			return err
		}
	}

	err := chConn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
				keyword String
		) Engine = TinyLog
	`, ignoredURLsTableName))

	if err != nil {
		err = &apiError{500, "Error occurred while creating ignored URLs table"}
	}
	return err
}

func fileReadIgnoredURLsThenWriteToTable(fileName string, ctx context.Context) error {

	file, err := os.Open(fileName)
	if err != nil {
		return &apiError{500, "Error occurred while opening ignored URLs file"}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	batch, err := chConn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", ignoredURLsTableName))
	if err != nil {
		return &apiError{500, "Error occurred while executing clickhouse query"}
	}
	for scanner.Scan() {
		err := batch.Append(scanner.Text())
		if err != nil {
			return &apiError{500, "Error occurred while executing clickhouse query"}
		}
	}

	return batch.Send()
}

func turnQueryParametersToMap(parametersWithDuplicate url.Values) map[string]string {
	allParameters := make(map[string]string)
	for k, v := range parametersWithDuplicate {
		allParameters[k] = v[0]
	}
	return allParameters
}
