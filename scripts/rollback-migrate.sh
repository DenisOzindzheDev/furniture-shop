#!/bin/bash
migrate -path migrations/ -database postgres://postgres:postgres@localhost:5432/furniture?sslmode=disable force 5