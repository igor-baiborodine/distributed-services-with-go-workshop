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

type ErrOffsetOutOfRange struct {
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

func (e ErrOffsetOutOfRange) Error() string {
	return e.ErrBookingLog.Error()
}

func NewErrOffsetOutOfRange(off uint64) *ErrOffsetOutOfRange {
	return &ErrOffsetOutOfRange{
		ErrBookingLog: &ErrBookingLog{
			Code: 404,
			ErrMsgFunc: func() string {
				return fmt.Sprintf("offset out of range: %d", off)
			},
		},
	}
}
