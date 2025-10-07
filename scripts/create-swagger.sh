#!/bin/bash

echo "Generating Swagger documentation..."

# Генерируем документацию
swag init -g cmd/api/main.go -o docs

if [ $? -eq 0 ]; then
    echo "Swagger documentation generated successfully!"
    echo "You can view it at: http://localhost:8080/swagger/index.html"
else
    echo "Failed to generate Swagger documentation"
    exit 1
fi