package app

import (
	"github.com/gin-gonic/gin"
	"log"
)

type ParseRequest struct {
	Url string
}

type App struct {
}

func (a *App) Run() {
	router := gin.Default()
	{
		router.POST("parse", func(context *gin.Context) {

		})

		router.GET("download/:fileId", func(context *gin.Context) {

		})
	}
	err := router.Run()
	if err != nil {
		log.Fatal(err)
	}
}
