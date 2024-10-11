package main

import (
	"log"
	"rental_service/auth"
	"rental_service/rent"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB() (*gorm.DB, error) {
	dsn := "host=localhost user=admin password=sahar223010 dbname=rental_service_db search_path=rent-request-service port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func NewLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

func NewValidator() *validator.Validate {
	return validator.New()
}

func RegisterRoutes(e *echo.Echo, handler *rent.RentHandler) {
	rentRequestGroup := e.Group("/rent-request")
	rentRequestGroup.Use(auth.AuthMiddleware)
	rentRequestGroup.POST("", handler.CreateRentRequest)
	rentRequestGroup.GET("/:rentRequestId", handler.GetRentRequestById)
	rentRequestGroup.PUT("/:rentRequestId/confirm", handler.ConfirmRentRequest)
	rentRequestGroup.PUT("/:rentRequestId/pay", handler.PayRentRequest)
	rentRequestGroup.PUT("/:rentRequestId/cancel", handler.CancelRentRequest)
	rentRequestGroup.GET("/owner", handler.GetOwnerRentRequests)
	rentRequestGroup.GET("/renter", handler.GetRenterRentRequests)
}

func main() {
	e := echo.New()

	app := fx.New(
		fx.Provide(
			NewDB,
			NewLogger,
			NewValidator,
			rent.NewRentRepository,
			rent.NewRentService,
			rent.NewRentHandler,
			func() *echo.Echo { return e },
		),
		fx.Invoke(
			func(e *echo.Echo, handler *rent.RentHandler) {
				RegisterRoutes(e, handler)
			},
			func() {
				if err := e.Start(":8082"); err != nil {
					log.Fatal("Echo server failed to start", zap.Error(err))
				}
			},
		),
	)
	app.Run()
}
