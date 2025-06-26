package models

import (
	"time"

	"gorm.io/gorm"
)

type Matter struct {
	gorm.Model
	Description               string    `gorm:"column:description"`
	OpenDate                  time.Time `gorm:"column:open_date"`
	CloseDate                 time.Time `gorm:"column:close_date"`
	PendingDate               time.Time `gorm:"column:pending_date"`
	LimitationDate            time.Time `gorm:"column:limitation_date"`
	IsBillable                bool      `gorm:"column:is_billable"`
	IsLimitationDateSatisfied bool      `gorm:"column:is_limitation_date_satisfied"`
	StatusID                  uint      `gorm:"column:status_id"`
	Rate                      float64   `gorm:"column:rate"`
	PracticeAreaID            uint      `gorm:"column:practice_area_id"`
	ClientID                  uint      `gorm:"column:client_id"`
	OriginatingAttorneyID     uint      `gorm:"column:originating_attorney_id"`
	ResponsibleAttorneyID     uint      `gorm:"column:responsible_attorney_id"`
	IsDeleted                 bool      `gorm:"column:is_deleted"`
	MatterNumber              string    `gorm:"column:matter_number"`
	Budget                    float64   `gorm:"column:budget"`
	HasBudget                 bool      `gorm:"column:has_budget"`
	Field1                    string    `gorm:"column:field1"`
	Field2                    string    `gorm:"column:field2"`
	Field3                    string    `gorm:"column:field3"`
	DisplayName               string    `gorm:"column:display_name"`
	CreatedByID               uint      `gorm:"column:created_by_id"`
	CreatedOn                 time.Time `gorm:"column:created_on"`
	ModifiedByID              uint      `gorm:"column:modified_by_id"`
	ModifiedOn                time.Time `gorm:"column:modified_on"`
	CustomFields              string    `gorm:"column:custom_fields"`
	CustomFormVersion         uint      `gorm:"column:custom_form_version"`
	RetainerFeeBillID         uint      `gorm:"column:retainer_fee_bill_id"`
	RetainerFeeFirstPayment   time.Time `gorm:"column:retainer_fee_first_payment"`
	RetainerFeeInitialAmount  float64   `gorm:"column:retainer_fee_initial_amount"`
	RetainerFeeLastBilledDate time.Time `gorm:"column:retainer_fee_last_billed_date"`
	RetainerFeeMonthlyAmount  float64   `gorm:"column:retainer_fee_monthly_amount"`
	RetainerFeeUserID         uint      `gorm:"column:retainer_fee_user_id"`
	FirmOfficeID              uint      `gorm:"column:firm_office_id"`
	SubjectAreaID             uint      `gorm:"column:subject_area_id"`
	IsHidden                  bool      `gorm:"column:is_hidden"`
	LawClerkID                uint      `gorm:"column:law_clerk_id"`
}
