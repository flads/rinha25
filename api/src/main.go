package main

import (
	"src/app"
)

func main() {
	appInstance := app.NewApp()

	appInstance.ListenAndServe()
}
