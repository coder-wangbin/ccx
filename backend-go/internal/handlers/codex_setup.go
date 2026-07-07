package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// CodexSetupResponse is the JSON returned by the /codex/setup endpoint.
type CodexSetupResponse struct {
	CatalogPath         string `json:"catalogPath"`
	CodexConfigSnippet  string `json:"codexConfigSnippet"`
	Instruction         string `json:"instruction"`
}

// CodexSetupHandler returns a handler that tells users how to configure Codex
// to use CCX's auto-generated model_catalog.json.
func CodexSetupHandler(stateDir string) gin.HandlerFunc {
	absPath, err := filepath.Abs(filepath.Join(stateDir, "model_catalog.json"))
	if err != nil {
		absPath = filepath.Join(stateDir, "model_catalog.json")
	}
	snippet := fmt.Sprintf("model_catalog_json = \"%s\"", absPath)

	return func(c *gin.Context) {
		c.JSON(http.StatusOK, CodexSetupResponse{
			CatalogPath:        absPath,
			CodexConfigSnippet: snippet,
			Instruction:        fmt.Sprintf("Add this line to ~/.codex/config.toml: %s", snippet),
		})
	}
}
