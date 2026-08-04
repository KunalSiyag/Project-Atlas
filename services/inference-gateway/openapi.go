package inference

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openAPIDocument []byte

// OpenAPIDocument returns the embedded OpenAPI 3 YAML for this service.
func OpenAPIDocument() []byte {
	return openAPIDocument
}

func (g *Gateway) handleOpenAPI(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	writer.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	writer.Header().Set("Cache-Control", "public, max-age=60")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(openAPIDocument)
}

func isOpenAPIPath(path string) bool {
	return path == "/openapi.yaml" || path == "/openapi.yml" || path == "/v1/openapi.yaml"
}
