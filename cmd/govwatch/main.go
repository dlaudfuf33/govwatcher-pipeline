package main

import (
	"github.com/joho/godotenv"

	"gwatch-data-pipeline/cmd"
	"gwatch-data-pipeline/internal/logging"
)

func main() {
	if err := godotenv.Load(); err != nil {
		logging.Warnf("Failed load .env file : %v", err)
	}

	cmd.Execute()
}
