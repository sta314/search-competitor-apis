package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/swaggo/gin-swagger"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	"google-competitor/api"
	_ "google-competitor/docs"
)

//@title Competitor Finder API
//@version 1.0
//@description This is a server that uses SERP database and returns possible competitors for a keyword
//@host localhost:8080
//@basePath /
func main() {
	router := gin.Default()
	router.GET("/get_competitors", api.RespondResults)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	err := router.Run("localhost:8080")

	if err != nil {
		fmt.Println("Gin failed to run")
		return
	}
}
