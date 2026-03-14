package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	outputDir := s.cfg.Output
	if strings.HasPrefix(outputDir, "~/") {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, outputDir[2:])
	}

	index := map[string][]string{}

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path) == "data.json" {
			domain := extractDomainFromPath(path, outputDir)
			index[domain] = append(index[domain], path)
		}
		return nil
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "could not read output directory"})
		return
	}

	json.NewEncoder(w).Encode(index)
}

func extractDomainFromPath(path, outputDir string) string {
	rel, _ := filepath.Rel(outputDir, path)
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}