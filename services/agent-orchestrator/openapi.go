package orchestrator

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

func isOpenAPIPath(path string) bool {
	return path == "/openapi.yaml" || path == "/openapi.yml" || path == "/v1/openapi.yaml"
}

func writeOpenAPI(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPIDocument)
}
