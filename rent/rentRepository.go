package rent

import (
	"time"

	"gorm.io/gorm"
)

type RentRequest struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RenterID      uint      `json:"renter_id"`
	OwnerID       uint      `json:"owner_id"`
	PostID        uint      `json:"post_id"`
	StartDate     time.Time `json:"start_date" validate:"required"`
	EndDate       time.Time `json:"end_date" validate:"required"`
	TotalPrice    int       `json:"total_price"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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

// func (rentRepo *RentRepository) GetRentRequestsByPostId(postId uint) ([]RentRequest, error) {
// 	var rentRequestList []RentRequest
// 	err := rentRepo.db.Where("post_id = ?", postId).Find(&rentRequestList).Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return rentRequestList, nil
// }

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
