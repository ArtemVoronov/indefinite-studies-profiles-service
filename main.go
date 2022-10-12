package main

import (
	"github.com/ArtemVoronov/indefinite-studies-profiles-service/internal/app"
)

func main() {
	app.Start()

	// TODO:
	// 1. update model: add uuid for user
	// 2. add sharding based on uuid
	// 3. send user uuid to auth service during verification of credentials
}
