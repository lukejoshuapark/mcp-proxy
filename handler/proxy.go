package handler

import "net/http"

func (s *Server) HandleProxy(w http.ResponseWriter, r *http.Request) {
	s.Proxy.ServeHTTP(w, r)
}
