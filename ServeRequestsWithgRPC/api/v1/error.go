package booking_v1

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type ErrBookingNotFound struct {
	UUID string
}

type ErrCreateBooking struct {
	Booking *Booking
}

type ErrUpdateBooking struct {
	Booking *Booking
}

func (e ErrBookingNotFound) GRPCStatus() *status.Status {
	msg := fmt.Sprintf("no booking found for UUID: %s", e.UUID)
	st := status.New(404, msg)

	d := &errdetails.LocalizedMessage{
		Locale:  "en-US",
		Message: msg,
	}
	std, err := st.WithDetails(d)
	if err != nil {
		return st
	}
	return std
}

func (e ErrBookingNotFound) Error() string {
	return e.GRPCStatus().Err().Error()
}

func (e ErrCreateBooking) GRPCStatus() *status.Status {
	msg := fmt.Sprintf("cannot create booking: %s", e.Booking)
	st := status.New(400, msg)

	d := &errdetails.LocalizedMessage{
		Locale:  "en-US",
		Message: msg,
	}
	std, err := st.WithDetails(d)
	if err != nil {
		return st
	}
	return std
}

func (e ErrCreateBooking) Error() string {
	return e.GRPCStatus().Err().Error()
}

func (e ErrUpdateBooking) GRPCStatus() *status.Status {
	msg := fmt.Sprintf("cannot update booking: %s", e.Booking)
	st := status.New(400, msg)

	d := &errdetails.LocalizedMessage{
		Locale:  "en-US",
		Message: msg,
	}
	std, err := st.WithDetails(d)
	if err != nil {
		return st
	}
	return std
}

func (e ErrUpdateBooking) Error() string {
	return e.GRPCStatus().Err().Error()
}
