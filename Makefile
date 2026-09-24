SHELL := /bin/bash

ENV_FILE ?= $(if $(wildcard .env),.env,backend/.env)
ENV_FILE_ABS := $(abspath $(ENV_FILE))
BACKEND_DIR := backend
BACKEND_CMD := go run . -f etc/chasing_points-api.yaml
PORT_WAIT_SECONDS ?= 30
BUILD_DIR := bin
BUILD_BINARY := $(BUILD_DIR)/chasing_points
RUN_DIR := .run
SERVER_LOCK_DIR := $(RUN_DIR)/server.lock

.PHONY: server stop restart build admin

admin:
	cd admin && npm run dev

server:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "Missing env file: $(ENV_FILE)"; \
		exit 1; \
	fi; \
	set -e; \
	read_env_var() { \
		local key="$$1"; \
		local value; \
		value=$$(awk -F= -v key="$$key" '\
			/^[[:space:]]*#/ { next } \
			{ \
				name=$$1; \
				gsub(/^[[:space:]]+|[[:space:]]+$$/, "", name); \
				if (name == key) { \
					sub(/^[^=]*=/, ""); \
					gsub(/^[[:space:]]+|[[:space:]]+$$/, ""); \
					print; \
					exit; \
				} \
			}' "$(ENV_FILE)"); \
		value="$${value%$$'\r'}"; \
		value="$${value#\"}"; \
		value="$${value%\"}"; \
		printf '%s' "$$value"; \
	}; \
	APP_PORT="$${APP_PORT:-$$(read_env_var APP_PORT)}"; \
	inherited_cf_tunnel_token="$$CF_TUNNEL_TOKEN"; \
	CF_TUNNEL_TOKEN="$$(read_env_var CF_TUNNEL_TOKEN)"; \
	CF_TUNNEL_TOKEN="$${CF_TUNNEL_TOKEN:-$$inherited_cf_tunnel_token}"; \
	if [ -z "$$APP_PORT" ]; then \
		echo "APP_PORT is missing in $(ENV_FILE)"; \
		exit 1; \
	fi; \
	if [ -z "$$CF_TUNNEL_TOKEN" ]; then \
		echo "CF_TUNNEL_TOKEN is missing in $(ENV_FILE) or current environment"; \
		exit 1; \
	fi; \
	mkdir -p "$(RUN_DIR)"; \
	if [ -e "$(SERVER_LOCK_DIR)" ]; then \
		server_pid=$$(cat "$(SERVER_LOCK_DIR)/server.pid" 2>/dev/null || true); \
		if [ -n "$$server_pid" ] && kill -0 "$$server_pid" 2>/dev/null; then \
			echo "Server is already running (PID $$server_pid). Use 'make restart' to restart it."; \
			exit 0; \
		fi; \
		echo "Removing stale server state"; \
		for pid_file in tunnel.pid backend-listener.pid backend.pid; do \
			pid=$$(cat "$(SERVER_LOCK_DIR)/$$pid_file" 2>/dev/null || true); \
			if [ -n "$$pid" ] && kill -0 "$$pid" 2>/dev/null; then \
				kill "$$pid" 2>/dev/null || true; \
			fi; \
		done; \
		rm -rf "$(SERVER_LOCK_DIR)"; \
	fi; \
	if ! mkdir "$(SERVER_LOCK_DIR)" 2>/dev/null; then \
		echo "Failed to acquire server lock: $(SERVER_LOCK_DIR)"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$$$" > "$(SERVER_LOCK_DIR)/server.pid"; \
	backend_pid=""; \
	backend_listener_pid=""; \
	tunnel_pid=""; \
	cleanup() { \
		trap - EXIT INT TERM HUP; \
		for pid in "$$tunnel_pid" "$$backend_listener_pid" "$$backend_pid"; do \
			if [ -n "$$pid" ]; then \
				kill "$$pid" 2>/dev/null || true; \
			fi; \
		done; \
		if [ -n "$$tunnel_pid" ]; then wait "$$tunnel_pid" 2>/dev/null || true; fi; \
		if [ -n "$$backend_pid" ]; then wait "$$backend_pid" 2>/dev/null || true; fi; \
		rm -rf "$(SERVER_LOCK_DIR)"; \
	}; \
	trap cleanup EXIT INT TERM HUP; \
	pids=$$(lsof -tiTCP:$$APP_PORT -sTCP:LISTEN 2>/dev/null || true); \
	if [ -n "$$pids" ]; then \
		echo "Port $$APP_PORT is in use, killing process(es): $$pids"; \
		kill $$pids 2>/dev/null || true; \
		for _ in 1 2 3 4 5; do \
			sleep 1; \
			pids=$$(lsof -tiTCP:$$APP_PORT -sTCP:LISTEN 2>/dev/null || true); \
			[ -z "$$pids" ] && break; \
		done; \
		pids=$$(lsof -tiTCP:$$APP_PORT -sTCP:LISTEN 2>/dev/null || true); \
		if [ -n "$$pids" ]; then \
			echo "Port $$APP_PORT is still in use, force killing process(es): $$pids"; \
			kill -9 $$pids 2>/dev/null || true; \
		fi; \
	fi; \
	echo "Starting backend: cd $(BACKEND_DIR) && $(BACKEND_CMD)"; \
	(cd $(BACKEND_DIR) && set -a && source "$(ENV_FILE_ABS)" && set +a && unset CF_TUNNEL_TOKEN && exec $(BACKEND_CMD)) & \
	backend_pid=$$!; \
	printf '%s\n' "$$backend_pid" > "$(SERVER_LOCK_DIR)/backend.pid"; \
	for _ in $$(seq 1 $(PORT_WAIT_SECONDS)); do \
		backend_listener_pid=$$(lsof -tiTCP:$$APP_PORT -sTCP:LISTEN 2>/dev/null | head -n 1 || true); \
		if [ -n "$$backend_listener_pid" ]; then \
			break; \
		fi; \
		if ! kill -0 "$$backend_pid" 2>/dev/null; then \
			wait "$$backend_pid"; \
			exit $$?; \
		fi; \
		sleep 1; \
	done; \
	if [ -z "$$backend_listener_pid" ]; then \
		echo "backend did not listen on port $$APP_PORT within $(PORT_WAIT_SECONDS) seconds"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$backend_listener_pid" > "$(SERVER_LOCK_DIR)/backend-listener.pid"; \
	echo "Starting cloudflared tunnel"; \
	(export TUNNEL_TOKEN="$$CF_TUNNEL_TOKEN"; unset CF_TUNNEL_TOKEN; exec cloudflared tunnel run) & \
	tunnel_pid=$$!; \
	printf '%s\n' "$$tunnel_pid" > "$(SERVER_LOCK_DIR)/tunnel.pid"; \
	wait "$$tunnel_pid"

stop:
	@if [ ! -e "$(SERVER_LOCK_DIR)" ]; then \
		echo "Server is not running"; \
		exit 0; \
	fi; \
	tunnel_pid=$$(cat "$(SERVER_LOCK_DIR)/tunnel.pid" 2>/dev/null || true); \
	backend_listener_pid=$$(cat "$(SERVER_LOCK_DIR)/backend-listener.pid" 2>/dev/null || true); \
	backend_pid=$$(cat "$(SERVER_LOCK_DIR)/backend.pid" 2>/dev/null || true); \
	server_pid=$$(cat "$(SERVER_LOCK_DIR)/server.pid" 2>/dev/null || true); \
	for pid in "$$tunnel_pid" "$$backend_listener_pid" "$$backend_pid" "$$server_pid"; do \
		if [ -n "$$pid" ] && kill -0 "$$pid" 2>/dev/null; then \
			kill "$$pid" 2>/dev/null || true; \
		fi; \
	done; \
	for _ in 1 2 3 4 5; do \
		running=""; \
		for pid in "$$tunnel_pid" "$$backend_listener_pid" "$$backend_pid" "$$server_pid"; do \
			if [ -n "$$pid" ] && kill -0 "$$pid" 2>/dev/null; then running=1; fi; \
		done; \
		[ -z "$$running" ] && break; \
		sleep 1; \
	done; \
	for pid in "$$tunnel_pid" "$$backend_listener_pid" "$$backend_pid" "$$server_pid"; do \
		if [ -n "$$pid" ] && kill -0 "$$pid" 2>/dev/null; then \
			kill -9 "$$pid" 2>/dev/null || true; \
		fi; \
	done; \
	rm -rf "$(SERVER_LOCK_DIR)"; \
	echo "Server stopped"

restart: stop
	@$(MAKE) server

build:
	@mkdir -p "$(BUILD_DIR)"
	@echo "Building $(BUILD_BINARY) for linux/amd64"
	@cd "$(BACKEND_DIR)" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "../$(BUILD_BINARY)" .
