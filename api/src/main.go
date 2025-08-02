package main

import (
	"log"
	"net/http"
	"src/app"
)

func main() {
	appInstance := app.NewApp()
	appInstance.SetupRoutes()

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
