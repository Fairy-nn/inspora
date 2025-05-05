package main

import (
	"github.com/gin-gonic/gin"

	events "github.com/Fairy-nn/inspora/internal/events/article"
)

type App struct {
	Server    *gin.Engine
	Consumers []events.Consumer
}
