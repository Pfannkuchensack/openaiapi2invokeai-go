package admin

import (
	"net/http"

	webadmin "github.com/Pfannkuchensack/openaiapi2invokeai-go/web/admin"
)

// StaticHandler returns an http.Handler serving the embedded static assets.
func StaticHandler() http.Handler {
	return http.FileServer(http.FS(webadmin.Static))
}
