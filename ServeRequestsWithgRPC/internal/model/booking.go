package model

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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
	CreatedAt time.Time
	UpdatedAt time.Time
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
		CreatedAt: timestamppb.New(b.CreatedAt),
		UpdatedAt: timestamppb.New(b.UpdatedAt),
	}
}

func (b Booking) String() string {
	return fmt.Sprintf("Booking(%s, %s, %s, %s, %s, %t, %v, %v)",
		b.UUID, b.Email, b.FullName, b.StartDate, b.EndDate, b.Active,
		b.CreatedAt, b.UpdatedAt)
}
