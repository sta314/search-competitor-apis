package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/swaggo/gin-swagger"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
	"google-suggestions/api"
	_ "google-suggestions/docs"
)

//@title Keyword Suggestion API
//@version 1.0
//@description This is a server that uses Google's autocomplete system to provide suggestions for a given keyword
//@host localhost:8080
//@basePath /

func main() {
	router := gin.Default()
	router.GET("/get_suggestions", api.RespondSuggestions)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	err := router.Run("localhost:8080")
	if err != nil {
		fmt.Println("Gin failed to run")
		return
	}
}
