package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync/atomic"
)

type Server struct {
	address       string
	port          string
	active_tokens atomic.Int64
}

func NewServer(ads string, po string) *Server {
	return &Server{
		address: ads,
		port:    po,
	}
}
func (s *Server) is_online() bool {
	return true
}
func (s *Server) Get_url_str() string {
	return fmt.Sprintf("http://%v:%v", s.address, s.port)
}

func tokenize(r *http.Request) int {

	body, err := io.ReadAll(r.Body)

	n := len(body)
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("couldnt tokenize %v", err)
	}

	return (n / 4)

}

type LoadBalance struct {
	servers []*Server
	RR      atomic.Int64
}

func (lb *LoadBalance) Add_Server(ser *Server) {
	lb.servers = append(lb.servers, ser)
}

func (lb *LoadBalance) Choose_Server() *Server {
	var best_serv *Server

	minTokens := int64(math.MaxInt64)
	for _, s := range lb.servers {
		if s.is_online() {
			curr_load := s.active_tokens.Load()
			if curr_load < minTokens {
				minTokens = curr_load
				best_serv = s

			}

		}
	}

	return best_serv

}
