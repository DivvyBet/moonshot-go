package api

import (
	"encoding/json"
	"net/http"

	"github.com/congruity7/moonshot-go/pkg/models"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// created by tadashi
func (c *Context) GetBetHistory(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var placedBets []models.PlacedBet

	c.ds.Db.Table("placed_bet").Find(&placedBets)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	bytes, _ := json.Marshal(placedBets)
	w.Write(bytes)

}

// created by tadashi
func (c *Context) CreateBetHistory(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var placedBet models.PlacedBet
	err := json.NewDecoder(r.Body).Decode(&placedBet)

	if err != nil {
		logrus.Error("creating wallet", err)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	result := c.ds.Db.Table("placed_bet").Create(&placedBet)

	if result.Error != nil {
		logrus.Error("creating placedBet", result.Error)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	bytes, _ := json.Marshal(placedBet)
	w.Write(bytes)
}
