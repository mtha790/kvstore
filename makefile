.PHONY: server run-server

# Variables for server and client
SERVER_DIR := server
SERVER_BIN := server-bin

# Build the server binary
server:
	@echo "Building server..."
	@go build -o $(SERVER_BIN) ./$(SERVER_DIR)/main.go

# Run the server
run-server: server
	@echo "Starting server..."
	@./$(SERVER_BIN)

