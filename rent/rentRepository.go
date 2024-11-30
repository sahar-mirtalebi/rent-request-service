package rent

import (
	"time"

	"gorm.io/gorm"
)

type RentRequest struct {
	ID            uint
	RenterID      uint
	OwnerID       uint
	PostID        uint
	StartDate     time.Time
	EndDate       time.Time
	TotalPrice    int
	Status        string
	PaymentStatus string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RentRepository struct {
	db *gorm.DB
}

func NewRentRepository(db *gorm.DB) *RentRepository {
	return &RentRepository{db: db}
}

func (rentRepo *RentRepository) AddRentRequest(rentRequest *RentRequest) error {
	return rentRepo.db.Create(&rentRequest).Error
}

func (rentRepo *RentRepository) UpdateRentRequest(rentRequest *RentRequest) error {
	return rentRepo.db.Save(&rentRequest).Error
}

func (rentRepo *RentRepository) GetOvelappingRequest(postId uint, status string, startDate, endDate time.Time) ([]RentRequest, error) {
	var rentRequestList []RentRequest
	err := rentRepo.db.Model(&RentRequest{}).Where("post_id = ? and status = ? and start_date <= ? and end_date >= ?", postId, status, endDate, startDate).Find(&rentRequestList).Error
	return rentRequestList, err
}

func (rentRepo *RentRepository) GetRentRequestsById(rentId uint) (*RentRequest, error) {
	var rentRequest RentRequest
	err := rentRepo.db.First(&rentRequest, rentId).Error
	if err != nil {
		return nil, err
	}
	return &rentRequest, nil
}

func (rentRepo *RentRepository) GetOwnerRentRequests(ownerId uint, status string, minDate, maxDate *time.Time, offset, limit int) ([]RentRequest, error) {
	var rentRequestList []RentRequest
	query := rentRepo.db.Model(&RentRequest{}).Where("owner_id = ?", ownerId)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if minDate != nil {
		query = query.Where("created_at >= ?", *minDate)
	}

	if maxDate != nil {
		query = query.Where("created_at <= ?", *maxDate)
	}

	err := query.Offset(offset).Limit(limit).Find(&rentRequestList).Error
	return rentRequestList, err
}

func (rentRepo *RentRepository) GetRenterRentRequests(renterId uint, status string, minDate, maxDate *time.Time, offset, limit int) ([]RentRequest, error) {
	var rentRequestList []RentRequest
	query := rentRepo.db.Model(&RentRequest{}).Where("renter_id = ?", renterId)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if minDate != nil {
		query = query.Where("created_at >= ?", *minDate)
	}

	if maxDate != nil {
		query = query.Where("created_at <= ?", *maxDate)
	}

	err := query.Offset(offset).Limit(limit).Find(&rentRequestList).Error
	return rentRequestList, err
}
