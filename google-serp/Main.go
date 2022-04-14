package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/swaggo/gin-swagger"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	"google-serp/api"
	_ "google-serp/docs"
)

//@title Google Serp API
//@version 1.0
//@description This is a server that parses Google's search results
//@host localhost:8080
//@basePath /

func main() {
	router := gin.Default()
	router.GET("/get_serp", api.RespondResults)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	err := router.Run("localhost:8080")
	if err != nil {
		fmt.Println("Gin failed to run")
		return
	}
}
	