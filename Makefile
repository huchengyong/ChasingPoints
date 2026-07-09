SHELL := /bin/bash

ENV_FILE ?= $(if $(wildcard .env),.env,backend/.env)
ENV_FILE_ABS := $(abspath $(ENV_FILE))
BACKEND_DIR := backend
BACKEND_CMD := go run chasing_points.go -f etc/chasing_points-api.yaml
PORT_WAIT_SECONDS ?= 30

.PHONY: server
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
	CF_TUNNEL_TOKEN="$${CF_TUNNEL_TOKEN:-$$(read_env_var CF_TUNNEL_TOKEN)}"; \
	if [ -z "$$APP_PORT" ]; then \
		echo "APP_PORT is missing in $(ENV_FILE)"; \
		exit 1; \
	fi; \
	if [ -z "$$CF_TUNNEL_TOKEN" ]; then \
		echo "CF_TUNNEL_TOKEN is missing in $(ENV_FILE) or current environment"; \
		exit 1; \
	fi; \
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
	(cd $(BACKEND_DIR) && set -a && source "$(ENV_FILE_ABS)" && set +a && exec $(BACKEND_CMD)) & \
	backend_pid=$$!; \
	cleanup() { kill $$backend_pid 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	for _ in $$(seq 1 $(PORT_WAIT_SECONDS)); do \
		if lsof -tiTCP:$$APP_PORT -sTCP:LISTEN >/dev/null 2>&1; then \
			break; \
		fi; \
		if ! kill -0 $$backend_pid 2>/dev/null; then \
			wait $$backend_pid; \
			exit $$?; \
		fi; \
		sleep 1; \
	done; \
	if ! lsof -tiTCP:$$APP_PORT -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "backend did not listen on port $$APP_PORT within $(PORT_WAIT_SECONDS) seconds"; \
		exit 1; \
	fi; \
	echo "Starting cloudflared tunnel"; \
	cloudflared tunnel run --token "$$CF_TUNNEL_TOKEN"
