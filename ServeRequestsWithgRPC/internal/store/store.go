package store

import (
	"fmt"
	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/model"
	"golang.org/x/exp/slices"
	"sync"
)

type BookingStore struct {
	mu       sync.Mutex
	bookings []model.Booking
}

func NewBookingStore() *BookingStore {
	return &BookingStore{}
}

//func (c *BookingStore) Create(booking model.Booking) (string, error) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	booking.UUID = uuid.New().String()
//	booking.Active = true
//	c.bookings = append(c.bookings, booking)
//	return booking.UUID, nil
//}

func (c *BookingStore) GetByUUID(uuid string) (model.Booking, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx := slices.IndexFunc(c.bookings, func(b model.Booking) bool { return b.UUID == uuid })
	if idx == -1 {
		return model.Booking{}, fmt.Errorf("booking not found")
	}
	return c.bookings[idx], nil
}
