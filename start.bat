@echo off
chcp 65001 >nul
echo ========================================
echo   ACFlow Labor Law RAG Search Service
echo ========================================
echo.
go run main.go -domains-dir data/domains -default-domain labor_law -min-score 0.3
