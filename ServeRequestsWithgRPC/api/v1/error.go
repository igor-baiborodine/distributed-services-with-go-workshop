package booking_v1

import (
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrBookingLog struct {
	Code       codes.Code
	ErrMsgFunc func() string
}

type ErrNotFoundForOffset struct {
	ErrBookingLog *ErrBookingLog
}

type ErrNotFoundForUUID struct {
	ErrBookingLog *ErrBookingLog
}

type ErrCreateBooking struct {
	ErrBookingLog *ErrBookingLog
}

type ErrUpdateBooking struct {
	ErrBookingLog *ErrBookingLog
}

func (e ErrBookingLog) GRPCStatus() *status.Status {
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

func (e ErrBookingLog) Error() string {
	return e.GRPCStatus().Err().Error()
}

func (e ErrNotFoundForOffset) Error() string {
	return e.ErrBookingLog.Error()
}

func (e ErrNotFoundForUUID) Error() string {
	return e.ErrBookingLog.Error()
}

func (e ErrCreateBooking) Error() string {
	return e.ErrBookingLog.Error()
}

func (e ErrUpdateBooking) Error() string {
	return e.ErrBookingLog.Error()
}

func NewErrNotFoundForOffset(off uint64) *ErrNotFoundForOffset {
	return &ErrNotFoundForOffset{
		ErrBookingLog: &ErrBookingLog{
			Code: 404,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("no booking found for offset: %d", off)
			},
		},
	}
}

func NewErrNotFoundForUUID(uuid string) *ErrNotFoundForUUID {
	return &ErrNotFoundForUUID{
		ErrBookingLog: &ErrBookingLog{
			Code: 404,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("no booking found for UUID: %s", uuid)
			},
		},
	}
}

func NewErrCreateBooking(b *Booking) *ErrCreateBooking {
	return &ErrCreateBooking{
		ErrBookingLog: &ErrBookingLog{
			Code: 400,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("cannot create booking: %s", b)
			},
		},
	}
}

func NewErrUpdateBooking(b *Booking) *ErrUpdateBooking {
	return &ErrUpdateBooking{
		ErrBookingLog: &ErrBookingLog{
			Code: 400,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("cannot update booking: %s", b)
			},
		},
	}
}
