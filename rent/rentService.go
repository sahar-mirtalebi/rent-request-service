package rent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RentService struct {
	repo   *RentRepository
	logger *zap.Logger
}

func NewRentService(repo *RentRepository, logger *zap.Logger) *RentService {
	return &RentService{repo: repo, logger: logger}
}

func (service *RentService) CreateRentRequest(renterID uint, rentRequest struct {
	PostId    uint      `json:"postId"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}) (uint, error) {

	rentRequestList, err := service.repo.GetOvelappingRequest(rentRequest.PostId, "Paid", rentRequest.StartDate, rentRequest.EndDate)
	if err != nil {
		service.logger.Error("error counting rent requests", zap.Error(err))
		return 0, echo.NewHTTPError(http.StatusInternalServerError, "error checking existing rent requests")
	}

	if len(rentRequestList) > 0 {
		return 0, echo.NewHTTPError(http.StatusConflict, "there is already a paid request in this period")
	}

	postDetail, err := GetPostByID(rentRequest.PostId)
	if err != nil {
		return 0, err
	}

	numberOfDays := int(rentRequest.EndDate.Sub(rentRequest.StartDate).Hours() / 24)
	if numberOfDays <= 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "einvalid date")
	}
	totalPrice := numberOfDays * int(postDetail.PricePerDay)

	newRentRequest := &RentRequest{
		RenterID:      renterID,
		OwnerID:       postDetail.OwnerId,
		PostID:        rentRequest.PostId,
		StartDate:     rentRequest.StartDate,
		EndDate:       rentRequest.EndDate,
		TotalPrice:    totalPrice,
		Status:        "waiting for confirmation",
		PaymentStatus: "pending",
		CreatedAt:     time.Now(),
	}
	err = service.repo.AddRentRequest(newRentRequest)
	if err != nil {
		service.logger.Error("Error creating rent request", zap.Error(err))
		return 0, echo.NewHTTPError(http.StatusInternalServerError, "failed to create rent request")
	}
	return newRentRequest.ID, nil
}

type PostResponseWithOwner struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PricePerDay float64 `json:"pricePerDay"`
	Address     string  `json:"address"`
	Category    string  `json:"category"`
	OwnerId     uint    `json:"ownerId"`
}

const GetPostByIdUrl = "http://localhost:8081/posts"

func GetPostByID(postId uint) (*PostResponseWithOwner, error) {
	url := fmt.Sprintf("%s/%d", GetPostByIdUrl, postId)
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post details : %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusBadRequest {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid post ID or bad request")
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, echo.NewHTTPError(http.StatusNotFound, "post not found")
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post not found or an error occurred: status code %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var postResponse PostResponseWithOwner
	if err := json.Unmarshal(body, &postResponse); err != nil {
		return nil, fmt.Errorf("failed to decode post response: %w", err)
	}

	return &postResponse, nil
}

type RentRequestResponse struct {
	StartDate     time.Time `json:"start_date" validate:"required"`
	EndDate       time.Time `json:"end_date" validate:"required"`
	TotalPrice    int       `json:"total_price"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
}

func (service *RentService) GetRentRequestById(userId, rentRequestId uint) (RentRequestResponse, error) {
	rentRequest, err := service.repo.GetRentRequestsById(rentRequestId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			service.logger.Error("error finding rentRequest", zap.Error(err))
			return RentRequestResponse{}, echo.NewHTTPError(http.StatusNotFound, "rentRequest not found")
		}
		service.logger.Error("error retrieving rentRequest", zap.Error(err))
		return RentRequestResponse{}, echo.NewHTTPError(http.StatusInternalServerError, "failed fo get rentRequest")
	}

	if userId != rentRequest.RenterID || userId != rentRequest.OwnerID {
		service.logger.Error("user ID does not match the rent request", zap.Error(err))
		return RentRequestResponse{}, echo.NewHTTPError(http.StatusForbidden, "user ID mismatch")
	}

	return RentRequestResponse{
		StartDate:     rentRequest.StartDate,
		EndDate:       rentRequest.EndDate,
		TotalPrice:    rentRequest.TotalPrice,
		Status:        rentRequest.Status,
		PaymentStatus: rentRequest.PaymentStatus,
	}, nil
}

func (service *RentService) ConfirmRentRequest(rentRequestId, ownerId uint) error {
	rentRequest, err := service.repo.GetRentRequestsById(rentRequestId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			service.logger.Error("error finding rentRequest", zap.Error(err))
			return echo.NewHTTPError(http.StatusNotFound, "rentRequest not found")
		}
		service.logger.Error("error retrieving rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed fo get rentRequest")
	}

	if rentRequest.OwnerID != ownerId {
		service.logger.Error("owner ID does not match the rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusForbidden, "owner ID mismatch")
	}

	if rentRequest.Status != "waiting for confirmation" {
		service.logger.Error("rent request is not in 'waiting for confirmation' state", zap.Error(err))
		return echo.NewHTTPError(http.StatusForbidden, "rent request cannot be confirmed as it is not in 'waiting for confirmation' state")
	}

	rentRequest.Status = "Confirmed"
	rentRequest.UpdatedAt = time.Now()

	err = service.repo.UpdateRentRequest(rentRequest)
	if err != nil {
		service.logger.Error("Error Updating rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update rent request")
	}
	return nil

}

func (service *RentService) PayRentRequest(renterId, rentRequestId uint) error {
	rentRequest, err := service.repo.GetRentRequestsById(rentRequestId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			service.logger.Error("error finding rentRequest", zap.Error(err))
			return echo.NewHTTPError(http.StatusNotFound, "rentRequest not found")
		}
		service.logger.Error("error retrieving rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed fo get rentRequest")
	}

	if rentRequest.RenterID != renterId {
		service.logger.Error("renter ID does not match the rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusForbidden, "renter ID mismatch")
	}

	if rentRequest.Status != "Confirmed" {
		service.logger.Error("rent request is not in 'Confirmed' state", zap.Error(err))
		return echo.NewHTTPError(http.StatusForbidden, "rent request cannot be paid as it is not in 'Confirmed' state")
	}

	rentRequest.Status = "Paid"
	rentRequest.PaymentStatus = "Success"
	rentRequest.UpdatedAt = time.Now()

	err = service.repo.UpdateRentRequest(rentRequest)
	if err != nil {
		service.logger.Error("Error Updating rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update rent request")
	}

	statuses := []string{"waiting for confirmation", "Confirmed"}
	postId := rentRequest.PostID
	startDate := rentRequest.StartDate
	endDate := rentRequest.EndDate
	for _, status := range statuses {
		rentRequestList, err := service.repo.GetOvelappingRequest(postId, status, startDate, endDate)
		if err != nil {
			service.logger.Error("error checking existing rent requests", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "error checking existing rent requests")
		}

		for _, overlappingRequest := range rentRequestList {
			overlappingRequest.Status = "Rejected"
			overlappingRequest.UpdatedAt = time.Now()
			err = service.repo.UpdateRentRequest(&overlappingRequest)
			if err != nil {
				service.logger.Error("Error rejecting overlaping rent request", zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to reject overlapping rent request")
			}
		}

	}

	return nil

}

func (service *RentService) CancelRentRequest(renterId, rentRequestId uint) error {
	rentRequest, err := service.repo.GetRentRequestsById(rentRequestId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			service.logger.Error("error finding rentRequest", zap.Error(err))
			return echo.NewHTTPError(http.StatusNotFound, "rentRequest not found")
		}
		service.logger.Error("error retrieving rentRequest", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed fo get rentRequest")
	}

	if rentRequest.RenterID != renterId {
		service.logger.Error("renter ID does not match the rent request", zap.Error(err))
		return echo.NewHTTPError(http.StatusForbidden, "renter ID mismatch")
	}

	if rentRequest.Status == "Confirmed" || rentRequest.Status == "waiting for confirmation" {
		rentRequest.Status = "canceled"
		rentRequest.UpdatedAt = time.Now()
		err := service.repo.UpdateRentRequest(rentRequest)
		if err != nil {
			service.logger.Error("Error canceling rent request", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to cancel rent request")
		}

	} else {
		service.logger.Error("rent request is not in 'Confirmed' or 'waiting for confirmation' state", zap.Error(err))
		return echo.NewHTTPError(http.StatusForbidden, "rent request cannot be paid as it is not in 'Confirmed' state")
	}
	return nil
}

func (service *RentService) GetOwnerRentRequests(ownerId uint, status string, minDate, maxDate *time.Time, page, size int) ([]RentRequestResponse, error) {
	offset := (page - 1) * size
	rents, err := service.repo.GetOwnerRentRequests(ownerId, status, minDate, maxDate, offset, size)
	if err != nil {
		service.logger.Error("fail getting rents", zap.Error(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "fail to fetch rents")
	}

	var rentResponseList []RentRequestResponse
	for _, rent := range rents {
		rentResponseList = append(rentResponseList, RentRequestResponse{
			StartDate:     rent.StartDate,
			EndDate:       rent.EndDate,
			TotalPrice:    rent.TotalPrice,
			Status:        rent.Status,
			PaymentStatus: rent.PaymentStatus,
		})
	}

	return rentResponseList, nil
}

func (service *RentService) GetRenterRentRequests(renterId uint, status string, minDate, maxDate *time.Time, page, size int) ([]RentRequestResponse, error) {
	offset := (page - 1) * size
	rents, err := service.repo.GetRenterRentRequests(renterId, status, minDate, maxDate, offset, size)
	if err != nil {
		service.logger.Error("fail getting rents", zap.Error(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "fail to fetch rents")
	}

	var rentResponseList []RentRequestResponse
	for _, rent := range rents {
		rentResponseList = append(rentResponseList, RentRequestResponse{
			StartDate:     rent.StartDate,
			EndDate:       rent.EndDate,
			TotalPrice:    rent.TotalPrice,
			Status:        rent.Status,
			PaymentStatus: rent.PaymentStatus,
		})
	}

	return rentResponseList, nil
}
