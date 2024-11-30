package rent

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type RentHandler struct {
	service *RentService

	validate *validator.Validate
}

func NewRentHandler(service *RentService, validate *validator.Validate) *RentHandler {
	return &RentHandler{service: service, validate: validate}
}

type RentDto struct {
	PostId    uint      `json:"postId" validate:"required"`
	StartDate time.Time `json:"startDate" validate:"required"`
	EndDate   time.Time `json:"endDate" validate:"required"`
}

func (handler *RentHandler) CreateRentRequest(c echo.Context) error {
	var rentRequest RentDto

	renterID, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if err := c.Bind(&rentRequest); err != nil {
		zap.L().Error("error binding request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "failed to bind request")
	}

	if err := handler.validate.Struct(rentRequest); err != nil {
		zap.L().Error("provided data is invalid", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid data")
	}

	createdRentRequest, err := handler.service.CreateRentRequest(renterID, rentRequest)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return c.JSON(http.StatusConflict, "there is already a paid request in this period")
		}
		zap.L().Error("error creating rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create rent request")
	}
	return c.JSON(http.StatusCreated, createdRentRequest)
}

func (handler *RentHandler) GetRentRequestById(c echo.Context) error {
	userId, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		zap.L().Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	rentRequest, err := handler.service.GetRentRequestById(userId, rentRequestIdStr)
	if err != nil {
		if errors.Is(ErrNotAllowed, err) {
			zap.L().Error("not allowed to retrieve post", zap.Error(err))
			return echo.NewHTTPError(http.StatusForbidden, "forbidden Access")
		}
		zap.L().Error("error retrieving rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve rent request")
	}

	return c.JSON(http.StatusOK, rentRequest)
}

func (handler *RentHandler) ConfirmRentRequest(c echo.Context) error {
	ownerID, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		zap.L().Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	err := handler.service.ConfirmRentRequest(rentRequestIdStr, ownerID)
	if err != nil {
		if errors.Is(ErrRecordNotFound, err) {
			zap.L().Error("error finding rentRequest", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to find rent request")
		} else if errors.Is(ErrNotAllowed, err) {
			zap.L().Error("not allowed to confirm rent request", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "forbidden Access")
		}
		zap.L().Error("error confirming rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to confirm rent request")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "the rent request has been confirmed successfully"})
}

func (handler *RentHandler) PayRentRequest(c echo.Context) error {
	renterId, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		zap.L().Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	redirectURL, err := handler.service.PayRentRequest(renterId, rentRequestIdStr)
	if err != nil {
		zap.L().Error("error retrieving redirectURL", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve redirectURL")
	}

	return c.JSON(http.StatusOK, map[string]string{"redirectURL": *redirectURL})
}

func (handler *RentHandler) UpdateRentRequestPaymentStatus(c echo.Context) error {

	rentRequestIdStr := c.QueryParam("requestId")
	status := c.QueryParam("status")
	if rentRequestIdStr == "" && status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "requestId and status are required")
	}

	message, err := handler.service.UpdateRentRequestPaymentStatus(rentRequestIdStr, status)
	if err != nil {
		zap.L().Error("error updating rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update rent request")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": *message})
}

func (handler *RentHandler) CancelRentRequest(c echo.Context) error {
	renterId, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	rentRequestIdStr := c.Param("rentRequestId")
	if rentRequestIdStr == "" {
		zap.L().Error("missed rentRequestId")
		return echo.NewHTTPError(http.StatusBadRequest, "rent-request ID is required")
	}

	err := handler.service.CancelRentRequest(renterId, rentRequestIdStr)
	if err != nil {
		zap.L().Error("error canceling rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to cancel rent request")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "the rent request has been canceled successfully"})
}

func (handler *RentHandler) GetOwnerRentRequests(c echo.Context) error {
	ownerId, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	status := c.QueryParam("status")
	dateStr := c.QueryParam("date")
	pageStr := c.QueryParam("page")

	rents, err := handler.service.GetOwnerRentRequests(ownerId, status, dateStr, pageStr)
	if err != nil {
		zap.L().Error("error getting rents", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch rents")
	}

	return c.JSON(http.StatusOK, rents)
}

func (handler *RentHandler) GetRenterRentRequests(c echo.Context) error {
	renterId, ok := c.Get("userId").(uint)
	if !ok {
		zap.L().Error("failed to get userId from context")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	status := c.QueryParam("status")
	dateStr := c.QueryParam("date")
	pageStr := c.QueryParam("page")

	rents, err := handler.service.GetRenterRentRequests(renterId, status, dateStr, pageStr)
	if err != nil {
		zap.L().Error("error getting rents", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch rents")
	}

	return c.JSON(http.StatusOK, rents)
}
