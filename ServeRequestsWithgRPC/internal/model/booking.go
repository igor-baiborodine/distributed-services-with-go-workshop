package model

import (
	api "github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/api/v1"
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
