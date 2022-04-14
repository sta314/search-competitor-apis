package api

import (
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"golang.org/x/net/context"
	"strings"
	"time"
)

const resultTableName = "google_serp_results"
const metadataTableName = "google_serp_results_meta"

var chConn driver.Conn = nil

//These are the columns that main table has

type tableResult struct {
	Keyword     string    `ch:"keyword"`
	Hl          string    `ch:"hl"`
	Gl          string    `ch:"gl"`
	Timestamp   time.Time `ch:"timestamp"`
	Url         string    `ch:"url"`
	Rank        uint8     `ch:"rank"`
	Title       string    `ch:"title"`
	Description string    `ch:"description"`
}

//These are the columns that metadata table has

type tableMeta struct {
	Keyword   string    `ch:"keyword"`
	Hl        string    `ch:"hl"`
	Gl        string    `ch:"gl"`
	Timestamp time.Time `ch:"timestamp"`
	Limit     uint8     `ch:"limit"`
}

func init() {
	err := initDB(context.Background())
	if err != nil {
		fmt.Println("Clickhouse connection failed")
	} else {
		fmt.Println("Clickhouse connection successful")
	}
}

//Make connection to DB and create tables if not already

func initDB(ctx context.Context) error {
	err := establishClickhouseConnection()
	if err != nil {
		return err
	}

	err = createTables(ctx, false)
	return err
}

//Establishes connection to DB

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
	}
	return err
}

//Returns whether DB is active and available

func isDBActive(ctx context.Context) bool {
	if chConn == nil {
		return false
	} else {
		if err := chConn.Ping(ctx); err != nil {
			return false
		}
	}

	return true
}

//Creates necessary tables

func createTables(ctx context.Context, dropIfExists bool) error {
	if dropIfExists {
		if err := chConn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", metadataTableName)); err != nil {
			return err
		}
		if err := chConn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", resultTableName)); err != nil {
			return err
		}
	}

	err := chConn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
				keyword String,
				hl String,
				gl String,
				timestamp Date,
				limit UInt8
		) Engine = MergeTree
		PARTITION BY keyword
		ORDER BY (hl, gl, timestamp)
	`, metadataTableName))

	if err != nil {
		return err
	}

	err = chConn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
				keyword String,
				hl String,
				gl String,
				timestamp Date,
				url String,
				rank UInt8,
				title String,
				description String
		) Engine = MergeTree
		PARTITION BY keyword
		ORDER BY (hl, gl, rank)
	`, resultTableName))

	return err
}

//Load results from the database using given parameters

func loadResultsFromDB(params searchParameters, ctx context.Context) ([]packedResult, error) {

	var rows []tableResult
	err := chConn.Select(ctx, &rows, fmt.Sprintf("SELECT * FROM %s WHERE keyword = '%s' AND hl = '%s' AND gl = '%s'", resultTableName, escapeQuote(params.keyword), params.hl, params.gl))
	if err != nil {
		return nil, err
	}

	var results []packedResult
	for _, row := range rows {
		result := packedResult{
			URL:         row.Url,
			Rank:        int(row.Rank),
			Title:       row.Title,
			Description: row.Description,
		}

		results = append(results, result)
	}

	return results, nil
}

//Add given results to the database

func saveResultsToDB(params searchParameters, results []packedResult, ctx context.Context) error {

	// clear old results if they exist
	oldRecordLimit, err := getLimitOfCachedResults(params, ctx)
	if err != nil {
		return err
	}

	if oldRecordLimit != -1 {
		err = removeResultsFromDB(params, ctx)
		if err != nil {
			return err
		}
	}

	curTime := time.Now()

	// insert metadata
	err = chConn.Exec(ctx, fmt.Sprintf(`INSERT INTO %s VALUES (
				'%s', '%s', '%s', toDate(%d), %d
			)`, metadataTableName, escapeQuote(params.keyword), params.hl, params.gl, curTime.Unix(), params.limit))
	if err != nil {
		return err
	}

	// insert data
	batch, errPrepare := chConn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", resultTableName))
	if errPrepare != nil {
		return errPrepare
	}
	for _, nextElement := range results {
		err := batch.AppendStruct(&tableResult{
			Keyword:     params.keyword,
			Hl:          params.hl,
			Gl:          params.gl,
			Timestamp:   curTime,
			Url:         nextElement.URL,
			Rank:        uint8(nextElement.Rank),
			Title:       nextElement.Title,
			Description: nextElement.Description,
		})
		if err != nil {
			return err
		}
	}

	fmt.Println("keyword done : " + params.keyword)
	return batch.Send()
}

//Remove existing results for given parameter

func removeResultsFromDB(params searchParameters, ctx context.Context) error {
	err := chConn.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s DELETE WHERE keyword = '%s' AND hl = '%s' AND gl = '%s'`, resultTableName, escapeQuote(params.keyword), params.hl, params.gl))
	if err != nil {
		return err
	}

	err = chConn.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s DELETE WHERE keyword = '%s' AND hl = '%s' AND gl = '%s'`, metadataTableName, escapeQuote(params.keyword), params.hl, params.gl))
	if err != nil {
		return err
	}

	return nil
}

//Check for existing results for given parameters and returns limit if they exist, return 0 if not

func getLimitOfCachedResults(params searchParameters, ctx context.Context) (int, error) {

	var metadata []tableMeta

	err := chConn.Select(ctx, &metadata, fmt.Sprintf("SELECT limit FROM %s WHERE keyword = '%s' AND hl = '%s' AND gl = '%s'", metadataTableName, escapeQuote(params.keyword), params.hl, params.gl))
	if err != nil {
		return -1, err
	}

	if len(metadata) > 0 {
		return int(metadata[0].Limit), nil
	}

	return -1, nil
}

//Used to escape single quotes for clickhouse

func escapeQuote(s string) string {
	return strings.Replace(s, "'", "''", -1)
}
