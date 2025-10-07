#!/bin/sh
curl -fsSL https://packagecloud.io/golang-migrate/migrate/gpgkey | sudo gpg --dearmor -o /etc/apt/keyrings/migrate.gpg
echo "deb [signed-by=/etc/apt/keyrings/migrate.gpg] https://packagecloud.io/golang-migrate/migrate/ubuntu/ $(lsb_release -sc) main" > /etc/apt/sources.list.d/migrate.list
apt-get update
apt-get install -y migrate