// Package server START: newhttpserver
package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func NewHTTPServer(addr string) *http.Server {
	httpsrv := newHTTPServer()
	r := mux.NewRouter()
	r.HandleFunc("/", httpsrv.handleCreate).Methods("POST")
	r.HandleFunc("/", httpsrv.handleRead).Methods("GET")
	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

// END: newhttpserver

// START: types
type httpServer struct {
	Campsite *Campsite
}

func newHTTPServer() *httpServer {
	return &httpServer{
		Campsite: NewCampsite(),
	}
}

type CreateRequest struct {
	Booking Booking `json:"booking"`
}

type CreateResponse struct {
	UUID string `json:"uuid"`
}

type ReadRequest struct {
	UUID string `json:"uuid"`
}

type ReadResponse struct {
	Booking Booking `json:"booking"`
}

// END:types

// START:create
func (s *httpServer) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uuid, err := s.Campsite.Create(req.Booking)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := CreateResponse{UUID: uuid}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// END:create

// START:read
func (s *httpServer) handleRead(w http.ResponseWriter, r *http.Request) {
	var req ReadRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	booking, err := s.Campsite.Read(req.UUID)
	if err == ErrBookingNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := ReadResponse{Booking: booking}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// END:read
