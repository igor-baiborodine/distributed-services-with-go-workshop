package store

import (
	"fmt"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/igor-baiborodine/distributed-services-with-go-workshop/SecureYourServices/AuthorizeWithAccessControlLists/internal/model"
)

type BookingStore struct {
	mu       sync.Mutex
	bookings []model.Booking
}

func NewBookingStore() (*BookingStore, error) {
	return &BookingStore{}, nil
}

func (s *BookingStore) GetByUUID(UUID string) (model.Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.reverseIndexFunc(func(b model.Booking) bool {
		return b.UUID == UUID
	})
	if idx == -1 {
		return model.Booking{}, fmt.Errorf("booking not found")
	}
	return s.bookings[idx], nil
}

func (s *BookingStore) GetByID(ID uint64) (model.Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.bookings,
		func(b model.Booking) bool { return b.ID == ID })
	if idx == -1 {
		return model.Booking{}, fmt.Errorf("booking not found")
	}
	return s.bookings[idx], nil
}

func (s *BookingStore) Create(b model.Booking) (model.Booking, error) {
	return s.appendBooking(b), nil
}

func (s *BookingStore) Update(b model.Booking) (model.Booking, error) {
	return s.appendBooking(b), nil
}

func (s *BookingStore) appendBooking(b model.Booking) model.Booking {
	s.mu.Lock()
	defer s.mu.Unlock()

	b.ID = uint64(len(s.bookings) + 1)
	s.bookings = append(s.bookings, b)
	return b
}

func (s *BookingStore) reverseIndexFunc(f func(model.Booking) bool) int {
	for i := len(s.bookings) - 1; i >= 0; i-- {
		if f(s.bookings[i]) {
			return i
		}
	}
	return -1
}
