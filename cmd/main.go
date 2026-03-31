package main

import (
	"bufio"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Application started")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			break
		}
		question := scanner.Text()

		answer := fmt.Sprintf("You asked: %s", question)
		logger.Info("Response", zap.String("answer", answer))
		fmt.Println("Response:", answer)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error I/O scanner", zap.Error(err))
	}
}
