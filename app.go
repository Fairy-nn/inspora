package main

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"

	events "github.com/Fairy-nn/inspora/internal/events/article"
)

type App struct {
	Server    *gin.Engine
	Consumers []events.Consumer
	Cron      *cron.Cron
}
