package models

import (
	"gorm.io/gorm"
)

type Candidates struct {
	Username string `gorm:"primarykey; not null; index:username,unique" json:"username"`

	FirstName    string `gorm:"not null" json:"first_name"`
	LastName     string `gorm:"not null" json:"last_name"`
	CountryPhone string
	Phone        string
	Address      string
	Latitude     string
	Longitude    string
	Visa         bool
	Description  string

	CurrentPosition string
	StartDate       int64

	WorkWhenever bool
	WorkShift    string

	ShareProfile bool
	ResumeLink   string
	ResumeName   string

	UpdatedAt int64 `gorm:"autoUpdateTime"`
	CreatedAt int64 `gorm:"autoCreateTime"`
}

func MigrateCandidates(db *gorm.DB) error {
	err := db.AutoMigrate(&Candidates{})
	return err
}
