package handlers

import (
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func dialogSaveRequest(r *http.Request) bool { return r.Header.Get("X-Search-Dialog") == "1" }

func writeDialogSave(w http.ResponseWriter, entryID primitive.ObjectID, bucket, locale string, existing bool, format string) {
	if locale != "en" && locale != "ja" {
		locale = "en"
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(struct {
		EntryID  string `json:"entryId"`
		Bucket   string `json:"bucket"`
		URL      string `json:"url"`
		Existing bool   `json:"existing"`
		Format   string `json:"format"`
	}{entryID.Hex(), bucket, "/" + locale + "/library/" + entryID.Hex(), existing, format})
}
