package rent

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type RentHandler struct {
	service  *RentService
	logger   *zap.Logger
	validate *validator.Validate
}

func NewRentHandler(service *RentService, logger *zap.Logger, validate *validator.Validate) *RentHandler {
	return &RentHandler{service: service, logger: logger, validate: validate}
}

func (handler *RentHandler) CreateRentRequest(c echo.Context) error {
	renterID, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var rentRequest struct {
		PostId    uint      `json:"postId"`
		StartDate time.Time `json:"startDate"`
		EndDate   time.Time `json:"endDate"`
	}
	if err := c.Bind(&rentRequest); err != nil {
		handler.logger.Error("error binding request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to bind request")
	}

	if err := handler.validate.Struct(rentRequest); err != nil {
		handler.logger.Error("provided data is invalid", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid data")
	}

	if !rentRequest.StartDate.Before(rentRequest.EndDate) {
		handler.logger.Error("provided data is invalid")
		return echo.NewHTTPError(http.StatusBadRequest, "invalid data")
	}

	createdRentRequest, err := handler.service.CreateRentRequest(renterID, rentRequest)
	if err != nil {
		if errors.Is(err, echo.ErrConflict) {
			return c.JSON(http.StatusConflict, "there is already a paid request in this period")
		}
		handler.logger.Error("error creating rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create rent request")
	}
	return c.JSON(http.StatusCreated, createdRentRequest)
}

func (handler *RentHandler) GetRentRequestById(c echo.Context) error {
	userId, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		handler.logger.Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		handler.logger.Error("invalid rentRequestId", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid rent ID")
	}

	rentRequest, err := handler.service.GetRentRequestById(userId, uint(rentRequestId))
	if err != nil {
		handler.logger.Error("error retrieving rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve rent request")
	}

	return c.JSON(http.StatusOK, rentRequest)
}

func (handler *RentHandler) ConfirmRentRequest(c echo.Context) error {
	ownerID, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		handler.logger.Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		handler.logger.Error("invalid rentRequestId", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid rent ID")
	}

	err = handler.service.ConfirmRentRequest(uint(rentRequestId), ownerID)
	if err != nil {
		handler.logger.Error("error confirming rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to confirm rent request")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "the rent request has been confirmed successfully"})
}

func (handler *RentHandler) PayRentRequest(c echo.Context) error {
	renterId, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		handler.logger.Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		handler.logger.Error("invalid rentRequestId", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid rent ID")
	}

	err = handler.service.PayRentRequest(renterId, uint(rentRequestId))
	if err != nil {
		handler.logger.Error("error paying rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to pay rent request")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "the rent request has been paid successfully"})
}

func (handler *RentHandler) CancelRentRequest(c echo.Context) error {
	renterId, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		handler.logger.Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		handler.logger.Error("invalid rentRequestId", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid rent ID")
	}

	err = handler.service.CancelRentRequest(renterId, uint(rentRequestId))
	if err != nil {
		handler.logger.Error("error canceling rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to cancel rent request")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "the rent request has been canceled successfully"})
}

func (handler *RentHandler) GetOwnerRentRequests(c echo.Context) error {
	ownerId, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	status := c.QueryParam("status")
	dateStr := c.QueryParam("date")
	var minDate, maxDate *time.Time
	if dateStr != "" {
		date := strings.Split(dateStr, ",")
		if date[0] != "" {
			min, err := time.Parse("2006-01-02", date[0])
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid minimum date")
			}
			minDate = &min
		}
		if date[1] != "" {
			max, err := time.Parse("2006-01-02", date[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid maximum date")
			}
			maxDate = &max
		}
		if minDate != nil && maxDate != nil {
			if minDate.After(*maxDate) {
				return echo.NewHTTPError(http.StatusBadRequest, "Minimum date cannot be greater than maximum date")
			}
		}
	}
	pageStr := c.QueryParam("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size := 10

	rents, err := handler.service.GetOwnerRentRequests(ownerId, status, minDate, maxDate, page, size)
	if err != nil {
		handler.logger.Error("error getting rents", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch rents")
	}

	return c.JSON(http.StatusOK, rents)
}

func (handler *RentHandler) GetRenterRentRequests(c echo.Context) error {
	renterId, ok := c.Get("userId").(uint)
	if !ok {
		handler.logger.Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	status := c.QueryParam("status")
	dateStr := c.QueryParam("date")
	var minDate, maxDate *time.Time
	if dateStr != "" {
		date := strings.Split(dateStr, ",")
		if date[0] != "" {
			min, err := time.Parse("2006-01-02", date[0])
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid minimum date")
			}
			minDate = &min
		}
		if date[1] != "" {
			max, err := time.Parse("2006-01-02", date[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid maximum date")
			}
			maxDate = &max
		}
		if minDate != nil && maxDate != nil {
			if minDate.After(*maxDate) {
				return echo.NewHTTPError(http.StatusBadRequest, "Minimum date cannot be greater than maximum date")
			}
		}
	}
	pageStr := c.QueryParam("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size := 10

	rents, err := handler.service.GetRenterRentRequests(renterId, status, minDate, maxDate, page, size)
	if err != nil {
		handler.logger.Error("error getting rents", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch rents")
	}

	return c.JSON(http.StatusOK, rents)
}
