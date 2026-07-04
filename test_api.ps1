# Test ACFlow RAG API

Write-Host "`n=== Testing ACFlow RAG API ===" -ForegroundColor Cyan

# Test 1: Status
Write-Host "`n[Test 1] Status Check" -ForegroundColor Yellow
$status = Invoke-RestMethod -Uri "http://localhost:9093/api/status"
Write-Host "Loaded: $($status.loaded)" -ForegroundColor Green
Write-Host "Documents: $($status.documents)" -ForegroundColor Green
Write-Host "Model: $($status.model)" -ForegroundColor Green

# Test 2: Search
Write-Host "`n[Test 2] Search Query" -ForegroundColor Yellow
$body = @{
    query = "加班工资怎么算"
    top_k = 5
} | ConvertTo-Json

$result = Invoke-RestMethod -Uri "http://localhost:9093/api/search" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

Write-Host "Query: $($result.query)" -ForegroundColor Green
Write-Host "Results: $($result.total)" -ForegroundColor Green

if ($result.results.Count -gt 0) {
    Write-Host "`nTop Result:" -ForegroundColor Cyan
    $top = $result.results[0]
    Write-Host "  Title: $($top.title)" -ForegroundColor White
    Write-Host "  Source: $($top.source)" -ForegroundColor White
    Write-Host "  Section: $($top.section)" -ForegroundColor White
    Write-Host "  Score: $($top.score)" -ForegroundColor White
    Write-Host "  Text: $($top.text.Substring(0, [Math]::Min(100, $top.text.Length)))..." -ForegroundColor Gray
} else {
    Write-Host "  No results found" -ForegroundColor Red
}

Write-Host "`n✅ Tests completed!" -ForegroundColor Green
