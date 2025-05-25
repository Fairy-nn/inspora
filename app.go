package main

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"

	"github.com/Fairy-nn/inspora/ioc"
)

type App struct {
	Server    *gin.Engine
	Consumers []ioc.Consumer
	// Consumers []events.Consumer
	Cron   *cron.Cron
	Search ioc.SearchInitializer
}
