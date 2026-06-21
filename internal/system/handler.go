package system

import (
	"net/http"
	"os"

	apihttp "go.naturallyfunny.dev/api/http"
)

type App struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Env     string `json:"env"`
}

var app = &App{
	Name:    "Zee",
	Version: "0.3.0",
	Env:     os.Getenv("APP_ENV"),
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	apihttp.WriteJSON(w, http.StatusOK, app)
}
