package main

import (
	"errors"
	"fmt"
	"github.com/brpaz/echozap"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	viper.SetConfigName("config")         // name of config file (without extension)
	viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
	viper.AddConfigPath(".")              // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
		} else {
			// Config file was found but another error was produced
		}
	}

	viper.SetDefault("PORT", "8080")

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

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", viper.GetInt("port"))))
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
