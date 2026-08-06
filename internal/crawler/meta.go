package crawler

import (
	"encoding/json"

	"github.com/kundanmergu/cloud-search-engine/ingestion/internal/models"
)

func decodeSeedMeta(raw []byte, meta *models.SeedMeta) error {
	return json.Unmarshal(raw, meta)
}
