package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/types"
	"gorm.io/gorm"
)

type StudioSSOTicket struct {
	Id            int            `json:"id"`
	TicketHash    string         `json:"ticket_hash" gorm:"type:char(64);uniqueIndex"`
	UserId        int            `json:"user_id" gorm:"index"`
	State         string         `json:"state" gorm:"type:varchar(128)"`
	Nonce         string         `json:"nonce" gorm:"type:varchar(128)"`
	ExpiresAt     int64          `json:"expires_at" gorm:"bigint;index"`
	UsedAt        *int64         `json:"used_at" gorm:"bigint;index"`
	IpHash        string         `json:"ip_hash" gorm:"type:char(64)"`
	UserAgentHash string         `json:"user_agent_hash" gorm:"type:char(64)"`
	CreatedAt     int64          `json:"created_at" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

type StudioImageReservation struct {
	Id                string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId            int    `json:"user_id" gorm:"index"`
	RequestId         string `json:"request_id" gorm:"type:varchar(128);index"`
	Provider          string `json:"provider" gorm:"type:varchar(64);index"`
	Model             string `json:"model" gorm:"type:varchar(128);index"`
	Size              string `json:"size" gorm:"type:varchar(32)"`
	Quality           string `json:"quality" gorm:"type:varchar(32)"`
	N                 int    `json:"n" gorm:"default:1"`
	EstimatedCost     int    `json:"estimated_cost"`
	FinalCost         int    `json:"final_cost"`
	Status            string `json:"status" gorm:"type:varchar(32);index"`
	UpstreamRef       string `json:"upstream_ref" gorm:"type:varchar(128)"`
	CredentialVersion int    `json:"credential_version" gorm:"default:1"`
	AllocationsJSON   string `json:"-" gorm:"type:text"`
	ExpiresAt         int64  `json:"expires_at" gorm:"bigint;index"`
	CommittedAt       int64  `json:"committed_at" gorm:"bigint"`
	ReleasedAt        int64  `json:"released_at" gorm:"bigint"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint"`
}

type StudioImageAuditLog struct {
	Id            int    `json:"id"`
	ReservationId string `json:"reservation_id" gorm:"type:varchar(64);index"`
	UserId        int    `json:"user_id" gorm:"index"`
	JobId         string `json:"job_id" gorm:"type:varchar(64);index"`
	EventType     string `json:"event_type" gorm:"type:varchar(64);index"`
	Provider      string `json:"provider" gorm:"type:varchar(64)"`
	Model         string `json:"model" gorm:"type:varchar(128)"`
	Amount        int    `json:"amount"`
	Status        string `json:"status" gorm:"type:varchar(32)"`
	ErrorCode     string `json:"error_code" gorm:"type:varchar(128)"`
	LatencyMs     int64  `json:"latency_ms"`
	MetadataJSON  string `json:"metadata_json" gorm:"type:text"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint"`
}

const (
	StudioImageReservationStatusReserved  = "reserved"
	StudioImageReservationStatusCommitted = "committed"
	StudioImageReservationStatusReleased  = "released"
)

func HashStudioSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func encodeStudioAllocations(allocations []types.QuotaFundingAllocation) string {
	data, _ := json.Marshal(allocations)
	return string(data)
}

func decodeStudioAllocations(data string) []types.QuotaFundingAllocation {
	if data == "" {
		return nil
	}
	var allocations []types.QuotaFundingAllocation
	_ = json.Unmarshal([]byte(data), &allocations)
	return allocations
}

func CreateStudioImageReservation(row *StudioImageReservation, allocations []types.QuotaFundingAllocation) error {
	now := time.Now().Unix()
	row.AllocationsJSON = encodeStudioAllocations(allocations)
	row.Status = StudioImageReservationStatusReserved
	row.CreatedAt = now
	row.UpdatedAt = now
	return DB.Create(row).Error
}

func GetStudioImageReservation(id string) (*StudioImageReservation, error) {
	var row StudioImageReservation
	err := DB.Where("id = ?", id).First(&row).Error
	return &row, err
}

func CommitStudioImageReservation(id string, finalCost int) (*StudioImageReservation, error) {
	row, err := GetStudioImageReservation(id)
	if err != nil {
		return nil, err
	}
	if row.Status == StudioImageReservationStatusCommitted {
		return row, nil
	}
	if row.Status != StudioImageReservationStatusReserved {
		return nil, errors.New("studio image reservation is not reserved")
	}
	if finalCost < 0 {
		finalCost = row.EstimatedCost
	}
	allocations := decodeStudioAllocations(row.AllocationsJSON)
	if finalCost > row.EstimatedCost {
		extraAllocations, err := ConsumeUserQuotaWithAllocation(row.UserId, finalCost-row.EstimatedCost)
		if err != nil {
			return nil, err
		}
		allocations = append(allocations, extraAllocations...)
	} else if finalCost < row.EstimatedCost {
		remaining, _, err := RefundUserQuotaAllocations(row.UserId, allocations, row.EstimatedCost-finalCost)
		if err != nil {
			return nil, err
		}
		allocations = remaining
	}
	now := time.Now().Unix()
	row.Status = StudioImageReservationStatusCommitted
	row.FinalCost = finalCost
	row.AllocationsJSON = encodeStudioAllocations(allocations)
	row.CommittedAt = now
	row.UpdatedAt = now
	return row, DB.Save(row).Error
}

func ReleaseStudioImageReservation(id string) (*StudioImageReservation, []types.QuotaFundingAllocation, error) {
	row, err := GetStudioImageReservation(id)
	if err != nil {
		return nil, nil, err
	}
	if row.Status == StudioImageReservationStatusReleased {
		return row, nil, nil
	}
	if row.Status != StudioImageReservationStatusReserved {
		return nil, nil, errors.New("studio image reservation is not reserved")
	}
	allocations := decodeStudioAllocations(row.AllocationsJSON)
	remaining, _, err := RefundUserQuotaAllocations(row.UserId, allocations, row.EstimatedCost)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().Unix()
	row.Status = StudioImageReservationStatusReleased
	row.ReleasedAt = now
	row.UpdatedAt = now
	row.AllocationsJSON = encodeStudioAllocations(remaining)
	return row, remaining, DB.Save(row).Error
}

func CreateStudioSSOTicket(ticket string, userId int, state string, nonce string, ttlSeconds int64, ip string, userAgent string) (*StudioSSOTicket, error) {
	now := time.Now().Unix()
	row := &StudioSSOTicket{
		TicketHash:    HashStudioSecret(ticket),
		UserId:        userId,
		State:         state,
		Nonce:         nonce,
		ExpiresAt:     now + ttlSeconds,
		IpHash:        HashStudioSecret(ip),
		UserAgentHash: HashStudioSecret(userAgent),
		CreatedAt:     now,
	}
	return row, DB.Create(row).Error
}

func ConsumeStudioSSOTicket(ticket string) (*StudioSSOTicket, error) {
	now := time.Now().Unix()
	ticketHash := HashStudioSecret(ticket)

	var row StudioSSOTicket
	err := DB.Where("ticket_hash = ?", ticketHash).First(&row).Error
	if err != nil {
		return nil, err
	}
	if row.UsedAt != nil {
		return nil, errors.New("studio sso ticket already used")
	}
	if row.ExpiresAt < now {
		return nil, errors.New("studio sso ticket expired")
	}

	err = DB.Model(&StudioSSOTicket{}).
		Where("id = ? AND used_at IS NULL", row.Id).
		Update("used_at", now).Error
	if err != nil {
		return nil, err
	}

	usedAt := now
	row.UsedAt = &usedAt
	return &row, nil
}
