package booking_v1

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrBookingNotFoundForID struct {
	ErrBooking *ErrBooking
}

type ErrBookingNotFoundForUUID struct {
	ErrBooking *ErrBooking
}

type ErrCreateBooking struct {
	ErrBooking *ErrBooking
}

type ErrUpdateBooking struct {
	ErrBooking *ErrBooking
}

type ErrBooking struct {
	Code       codes.Code
	ErrMsgFunc func() string
}

func (e ErrBooking) GRPCStatus() *status.Status {
	msg := e.ErrMsgFunc()
	st := status.New(e.Code, msg)

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

func (e ErrBooking) Error() string {
	return e.GRPCStatus().Err().Error()
}

func NewErrBookingNotFoundForID(id int) *ErrBookingNotFoundForID {
	return &ErrBookingNotFoundForID{
		ErrBooking: &ErrBooking{
			Code: 404,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("no booking found for ID: %d", id)
			},
		},
	}
}

func NewErrBookingNotFoundForUUID(uuid string) *ErrBookingNotFoundForUUID {
	return &ErrBookingNotFoundForUUID{
		ErrBooking: &ErrBooking{
			Code: 404,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("no booking found for UUID: %s", uuid)
			},
		},
	}
}

func NewErrCreateBooking(b *Booking) *ErrCreateBooking {
	return &ErrCreateBooking{
		ErrBooking: &ErrBooking{
			Code: 400,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("cannot create booking: %s", b)
			},
		},
	}
}

func NewErrUpdateBooking(b *Booking) *ErrUpdateBooking {
	return &ErrUpdateBooking{
		ErrBooking: &ErrBooking{
			Code: 400,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("cannot update booking: %s", b)
			},
		},
	}
}
