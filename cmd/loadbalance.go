package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
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

type LoadBalance struct {
	servers []*Server
	RR      atomic.Int64
	Rad_T   *Tree
	mu      sync.Mutex
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

func (lb *LoadBalance) GetInfo(r *http.Request) Body_Info {

	body, err := io.ReadAll(r.Body)

	n := len(body)
	r.Body = io.NopCloser(bytes.NewBuffer(body)) // normal reader is wrong, as once read the body is cleaned out
	if err != nil {
		fmt.Printf("couldnt tokenize %v", err)
	}
	prompt := string(body)
	lb.mu.Lock()
	retnode, server_ := lb.Rad_T.Insert(prompt)
	lb.mu.Unlock()
	return Body_Info{retnode: retnode, server_: server_, token: n / 4}

}

func (lb *LoadBalance) Picker(lat_server *Server, cached_server *Server) *Server {

	if cached_server != nil {
		lat_tok := lat_server.active_tokens.Load()
		cached_tok := cached_server.active_tokens.Load()

		if cached_tok-lat_tok <= 2000 {
			return cached_server
		}
	}
	return lat_server

}
