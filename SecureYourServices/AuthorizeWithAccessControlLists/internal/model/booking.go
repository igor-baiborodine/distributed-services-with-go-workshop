package model

import (
	"fmt"

	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/api/v1"
)

// Booking model
type Booking struct {
	UUID      string
	Email     string
	FullName  string
	StartDate string
	EndDate   string
	Active    bool
}

// ProtoBooking creates booking proto from Booking model
func (b *Booking) ProtoBooking() *api.Booking {
	return &api.Booking{
		UUID:      b.UUID,
		Email:     b.Email,
		FullName:  b.FullName,
		StartDate: b.StartDate,
		EndDate:   b.EndDate,
		Active:    b.Active,
	}
}

func (b Booking) String() string {
	return fmt.Sprintf("Booking(%s, %s, %s, %s, %s, %t)",
		b.UUID, b.Email, b.FullName, b.StartDate, b.EndDate, b.Active)
}
