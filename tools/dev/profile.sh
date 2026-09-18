#!/usr/bin/env bash
# githint-profile.sh — roda com daemon já ativo, usa por ~10s enquanto isso
echo "Coletando 10s de CPU profile... usa o terminal normalmente agora."
go tool pprof -top -seconds=10 http://localhost:6060/debug/pprof/profile