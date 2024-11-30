package rent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type RentService struct {
	repo *RentRepository
}

func NewRentService(repo *RentRepository) *RentService {
	return &RentService{repo: repo}
}

var ErrConflict = errors.New("there is alreay a paid request for this period")
var ErrRecordNotFound = errors.New("rentRequest not found")
var ErrNotAllowed = errors.New("owner ID mismatch")

func (service *RentService) CreateRentRequest(renterID uint, rentRequest RentDto) (*uint, error) {
	if !rentRequest.StartDate.Before(rentRequest.EndDate) {
		return nil, fmt.Errorf("invalid date")
	}

	rentRequestList, err := service.repo.GetOvelappingRequest(rentRequest.PostId, "Paid", rentRequest.StartDate, rentRequest.EndDate)
	if err != nil {
		return nil, err
	}

	if len(rentRequestList) > 0 {
		return nil, ErrConflict
	}

	postDetail, err := GetPostByID(rentRequest.PostId)
	if err != nil {
		return nil, err
	}

	numberOfDays := int(rentRequest.EndDate.Sub(rentRequest.StartDate).Hours() / 24)
	if numberOfDays <= 0 {
		return nil, fmt.Errorf("bad request")
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

		return nil, err
	}
	return &newRentRequest.ID, nil
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
		return nil, fmt.Errorf("invalid post ID or bad request")
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("post not found")
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

func (service *RentService) GetRentRequestById(userId uint, rentRequestIdStr string) (*RentRequestResponse, error) {
	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		return nil, err
	}
	rentRequest, err := service.repo.GetRentRequestsById(uint(rentRequestId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		return nil, err
	}

	if userId != rentRequest.RenterID || userId != rentRequest.OwnerID {
		return nil, ErrNotAllowed
	}

	return &RentRequestResponse{
		StartDate:     rentRequest.StartDate,
		EndDate:       rentRequest.EndDate,
		TotalPrice:    rentRequest.TotalPrice,
		Status:        rentRequest.Status,
		PaymentStatus: rentRequest.PaymentStatus,
	}, nil
}

func (service *RentService) ConfirmRentRequest(rentRequestIdStr string, ownerId uint) error {
	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		return err
	}
	rentRequest, err := service.repo.GetRentRequestsById(uint(rentRequestId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	if rentRequest.OwnerID != ownerId {
		return ErrNotAllowed
	}

	if rentRequest.Status != "waiting for confirmation" {
		return ErrNotAllowed
	}

	rentRequest.Status = "Confirmed"
	rentRequest.UpdatedAt = time.Now()

	err = service.repo.UpdateRentRequest(rentRequest)
	if err != nil {
		return err
	}
	return nil

}

func (service *RentService) PayRentRequest(renterId uint, rentRequestIdStr string) (*string, error) {
	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		return nil, err
	}

	rentRequest, err := service.repo.GetRentRequestsById(uint(rentRequestId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	if rentRequest.RenterID != renterId {
		return nil, ErrNotAllowed
	}

	if rentRequest.Status != "Confirmed" {
		return nil, ErrNotAllowed
	}

	paymentPayload := map[string]interface{}{
		"requestId":   rentRequest.ID,
		"amount":      rentRequest.TotalPrice,
		"callbackURL": fmt.Sprintf("http://localhost:8082/rent-request/callback?requestId=%v", rentRequest.ID),
	}

	redirectURL, err := service.CreatePaymentRequest(paymentPayload)
	if err != nil {
		return nil, err
	}

	return redirectURL, nil
}

func (service *RentService) CreatePaymentRequest(paymentPayload map[string]interface{}) (*string, error) {
	requestBody, err := json.Marshal(paymentPayload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", "http://localhost:8083/payment/request", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("an error occurred: status code %d", response.StatusCode)
	}

	var result struct {
		RedirectURL string `json:"redirectURL"`
	}

	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.RedirectURL, nil
}

func (service *RentService) UpdateRentRequestPaymentStatus(rentRequestIdStr, status string) (*string, error) {
	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		return nil, err
	}
	rentRequest, err := service.repo.GetRentRequestsById(uint(rentRequestId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	if rentRequest.PaymentStatus == "success" {
		return nil, err
	}

	if rentRequest.Status != "Confirmed" {
		return nil, err
	}

	if status != "success" && status != "cancel" {
		return nil, err
	}

	rentRequest.PaymentStatus = status
	rentRequest.UpdatedAt = time.Now()

	if status == "success" {
		rentRequest.Status = "paid"
		err = service.repo.UpdateRentRequest(rentRequest)
		if err != nil {
			return nil, err
		}

		states := []string{"waiting for confirmation", "Confirmed"}
		postId := rentRequest.PostID
		startDate := rentRequest.StartDate
		endDate := rentRequest.EndDate
		for _, state := range states {
			rentRequestList, err := service.repo.GetOvelappingRequest(postId, state, startDate, endDate)
			if err != nil {
				return nil, err
			}

			for _, overlappingRequest := range rentRequestList {
				if overlappingRequest.ID != uint(rentRequestId) {
					overlappingRequest.Status = "Rejected"
					overlappingRequest.UpdatedAt = time.Now()
					err = service.repo.UpdateRentRequest(&overlappingRequest)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		message := "Your payment was processed successfully!"

		return &message, nil
	}
	err = service.repo.UpdateRentRequest(rentRequest)
	if err != nil {
		return nil, err
	}
	message := "Your payment has been canceled"
	return &message, nil
}

func (service *RentService) CancelRentRequest(renterId uint, rentRequestIdStr string) error {
	rentRequestId, err := strconv.ParseUint(rentRequestIdStr, 10, 32)
	if err != nil {
		return err
	}

	rentRequest, err := service.repo.GetRentRequestsById(uint(rentRequestId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	if rentRequest.RenterID != renterId {
		return err
	}

	if rentRequest.Status == "Confirmed" || rentRequest.Status == "waiting for confirmation" {
		rentRequest.Status = "canceled"
		rentRequest.UpdatedAt = time.Now()
		err := service.repo.UpdateRentRequest(rentRequest)
		if err != nil {
			return err
		}

	} else {
		return err
	}
	return nil
}

func (service *RentService) GetOwnerRentRequests(ownerId uint, status, dateStr, pageStr string) ([]RentRequestResponse, error) {
	var minDate, maxDate *time.Time
	if dateStr != "" {
		date := strings.Split(dateStr, ",")
		if date[0] != "" {
			min, err := time.Parse("2006-01-02", date[0])
			if err != nil {
				return nil, err
			}
			minDate = &min
		}
		if date[1] != "" {
			max, err := time.Parse("2006-01-02", date[1])
			if err != nil {
				return nil, err
			}
			maxDate = &max
		}
		if minDate != nil && maxDate != nil {
			if minDate.After(*maxDate) {
				return nil, fmt.Errorf("invalid date")
			}
		}
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size := 10

	offset := (page - 1) * size
	rents, err := service.repo.GetOwnerRentRequests(ownerId, status, minDate, maxDate, offset, size)
	if err != nil {
		return nil, err
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

func (service *RentService) GetRenterRentRequests(renterId uint, status, dateStr, pageStr string) ([]RentRequestResponse, error) {
	var minDate, maxDate *time.Time
	if dateStr != "" {
		date := strings.Split(dateStr, ",")
		if date[0] != "" {
			min, err := time.Parse("2006-01-02", date[0])
			if err != nil {
				return nil, err
			}
			minDate = &min
		}
		if date[1] != "" {
			max, err := time.Parse("2006-01-02", date[1])
			if err != nil {
				return nil, err
			}
			maxDate = &max
		}
		if minDate != nil && maxDate != nil {
			if minDate.After(*maxDate) {
				return nil, fmt.Errorf("invalid date")
			}
		}
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size := 10
	offset := (page - 1) * size
	rents, err := service.repo.GetRenterRentRequests(renterId, status, minDate, maxDate, offset, size)
	if err != nil {
		return nil, err
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
