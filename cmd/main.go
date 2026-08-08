package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Middleware struct {
	h  *httputil.ReverseProxy
	lb *LoadBalance
}

func (middle *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	New_req := r.WithContext(context.WithValue(r.Context(), "Body_Info", middle.lb.GetInfo(r)))

	middle.h.ServeHTTP(w, New_req)
}

type Body_Info struct {
	token   int
	retnode *node
	server_ *Server
}

func main() {

	lb := LoadBalance{Rad_T: &Tree{}}
	lb.Add_Server(NewServer("localhost", "8081"))
	lb.Add_Server(NewServer("localhost", "8082"))
	lb.Add_Server(NewServer("localhost", "8083"))

	mux := http.NewServeMux()

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			lat_server := lb.Choose_Server()
			un_Info := r.In.Context().Value("Body_Info")
			Info, ok := un_Info.(Body_Info)
			if ok {
				selected_server := lb.Picker(lat_server, Info.server_)
				selected_server.active_tokens.Add(int64(Info.token))
				r.Out = r.Out.WithContext(context.WithValue(r.Out.Context(), "s_server", selected_server))

				lb.mu.Lock()
				Info.retnode.server = selected_server // assigning the selected server to the matched node
				lb.mu.Unlock()

				server_url, _ := url.Parse(selected_server.Get_url_str())
				r.SetURL(server_url)
				r.SetXForwarded()
			}
		}, ModifyResponse: func(res *http.Response) error {
			un_Info := res.Request.Context().Value("Body_Info")

			Info, _ := un_Info.(Body_Info)

			temp_val := res.Request.Context().Value("s_server")

			s_server, _ := temp_val.(*Server)
			s_server.active_tokens.Add(-int64(Info.token))

			return nil
		}, ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if temp_val := r.Context().Value("s_server"); temp_val != nil {
				if s_server, ok := temp_val.(*Server); ok {
					if Info := r.Context().Value("Body_Info"); Info != nil {
						s_server.active_tokens.Add(-int64(Info.(Body_Info).token))
					}
				}
			}
			w.WriteHeader(http.StatusBadGateway)
		},
	}
	mid_prox := Middleware{h: proxy, lb: &lb}
	mux.Handle("/", &mid_prox)

	err := http.ListenAndServe(":9090", mux)
	if err == http.ErrServerClosed {
		fmt.Println("Server is closed")
	} else if err != nil {
		s := fmt.Sprintf("err occured %v", err)
		fmt.Println(s)
	}

}
