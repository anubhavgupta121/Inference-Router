package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Middleware struct {
	h *httputil.ReverseProxy
}

func (middle *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Tokens := tokenize(r)
	New_req := r.Clone(context.WithValue(r.Context(), "token", Tokens))

	middle.h.ServeHTTP(w, New_req)
}

func main() {

	lb := LoadBalance{}
	lb.Add_Server(NewServer("localhost", "8081"))
	lb.Add_Server(NewServer("localhost", "8082"))
	lb.Add_Server(NewServer("localhost", "8083"))

	mux := http.NewServeMux()

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			selected_server := lb.Choose_Server()
			server_url, _ := url.Parse(selected_server.Get_url_str())

			Value := r.In.Context().Value("token")

			Tokens, ok := Value.(int)
			if ok {
				selected_server.active_tokens.Add(int64(Tokens))
				r.Out = r.Out.Clone(context.WithValue(r.Out.Context(), "server", selected_server))
				r.SetURL(server_url)
				r.SetXForwarded()
			}
		}, ModifyResponse: func(res *http.Response) error {
			temp_val := res.Request.Context().Value("token")

			Tokens, _ := temp_val.(int)

			temp_val = res.Request.Context().Value("server")

			s_server, _ := temp_val.(*Server)
			s_server.active_tokens.Add(-int64(Tokens))

			return nil
		}, ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if temp_val := r.Context().Value("server"); temp_val != nil {
				if s_server, ok := temp_val.(*Server); ok {
					if Tokens := r.Context().Value("token"); Tokens != nil {
						s_server.active_tokens.Add(-int64(Tokens.(int)))
					}
				}
			}
			w.WriteHeader(http.StatusBadGateway)
		},
	}
	mid_prox := Middleware{h: proxy}
	mux.Handle("/", &mid_prox)

	err := http.ListenAndServe(":9090", mux)
	if err == http.ErrServerClosed {
		fmt.Println("Server is closed")
	} else if err != nil {
		s := fmt.Sprintf("err occured %v", err)
		fmt.Println(s)
	}

}
