package store

import (
	"fmt"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/igor-baiborodine/distributed-services-with-go-workshop/ServeRequestsWithgRPC/internal/model"
)

type BookingStore struct {
	mu       sync.Mutex
	bookings []model.Booking
}

func NewBookingStore() (*BookingStore, error) {
	return &BookingStore{}, nil
}

func (s *BookingStore) GetByUUID(uuid string) (model.Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.bookings, func(b model.Booking) bool {
		return b.UUID == uuid
	})
	if idx == -1 {
		return model.Booking{}, fmt.Errorf("booking not found")
	}
	return s.bookings[idx], nil
}

func (s *BookingStore) Create(b model.Booking) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bookings = append(s.bookings, b)
	return nil
}

func (s *BookingStore) Update(b model.Booking) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.bookings,
		func(eb model.Booking) bool { return eb.UUID == b.UUID })
	s.bookings[idx] = b
	return nil
}
