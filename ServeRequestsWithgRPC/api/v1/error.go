package booking_v1

import (
	"fmt"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type ErrBookingNotFound struct {
	UUID string
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
