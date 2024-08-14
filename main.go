package main

import (
	"errors"
	"fmt"
	"github.com/brpaz/echozap"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(echozap.ZapLogger(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(otelecho.Middleware("sticker_track"))
	e.Use(echoprometheus.NewMiddleware("sticker_track")) // adds middleware to gather metrics
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/sticker", sticker)
	pprof.Register(e)

	e.Logger.Fatal(e.Start(":1323"))
}

func sticker(c echo.Context) error {
	slotId := c.QueryParam("slot_id")

	if slotId == "" {
		err := c.NoContent(http.StatusBadRequest)
		if err != nil {
			return err
		}
		return nil
	}

	file, err := os.OpenFile("sticker.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		zap.Error(err)
		err2 := c.NoContent(http.StatusInternalServerError)
		if err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}
	line := fmt.Sprintf("%v,%v\n", time.Now().Unix(), slotId)
	_, err = file.Write([]byte(line))
	if err != nil {
		zap.Error(err)
		err2 := c.NoContent(http.StatusInternalServerError)
		if err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}
	err = file.Close()
	if err != nil {
		zap.Error(err)
		err2 := c.NoContent(http.StatusInternalServerError)
		if err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}
	err = c.NoContent(http.StatusOK)
	if err != nil {
		return err
	}
	return nil
}
