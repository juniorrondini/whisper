if (!(Test-Path ".env")) {
  Copy-Item ".env.example" ".env"
}

Write-Host "Environment file ready. Starting Docker Compose..."
docker compose up --build
