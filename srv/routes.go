package srv

import (
	"log"
	"net/http"

	"github.com/im-karina/basics/cfg"
)

func Serve(_ string) error {
	mux := http.NewServeMux()

	log.Println("listening on:", cfg.ListenAddr)
	return http.ListenAndServe(cfg.ListenAddr, mux)
}
