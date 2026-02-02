package main

import (
	"doorman/internal/app"
	"doorman/internal/config"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	api := r.Group("/api/v1")

	api.POST("/otp/send", application.OtpHandler.Send)
	api.POST("/otp/verify", application.OtpHandler.Verify)

	api.POST("/auth/register", application.RegistrationHandler.Register)

	api.POST("/auth/refresh", application.TokenHandler.Refresh)
	api.POST("/auth/logout", application.TokenHandler.Logout)

	r.GET("/.well-known/jwks.json", application.KeysHandler.GetJWKS)

	err = r.Run(fmt.Sprintf(":%d", cfg.App.Port))
	if err != nil {
		log.Fatal(err)
	}
}
