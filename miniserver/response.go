package miniserver

import (
	"net/http"
)

type Response interface {
	write(writer http.ResponseWriter, client *Client)
}
