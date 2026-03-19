<#
    Einmalige Auth-Einrichtung fuer Claude Code im Container
    
    Erstellt ein persistentes Podman-Volume und startet Claude Code
    interaktiv zum Einloggen. Danach bleiben die Credentials gespeichert.
#>

param(
    [string]$ImageName = "claude-swarmsim",
    [string]$VolumeName = "claude-auth"
)

Write-Host ""
Write-Host "=== Claude Code Auth Setup ===" -ForegroundColor Cyan
Write-Host ""

# Prüfe ob Image existiert
$imageExists = podman images -q $ImageName 2>$null
if (-not $imageExists) {
    Write-Host "[!] Image '$ImageName' nicht gefunden." -ForegroundColor Red
    Write-Host "Erst bauen: podman build -f Dockerfile -t $ImageName ." -ForegroundColor Yellow
    exit 1
}

# Erstelle persistentes Volume (falls noch nicht vorhanden)
$volumeExists = podman volume inspect $VolumeName 2>$null
if (-not $volumeExists) {
    Write-Host "[>] Erstelle persistentes Volume '$VolumeName'..." -ForegroundColor Cyan
    podman volume create $VolumeName
} else {
    Write-Host "[+] Volume '$VolumeName' existiert bereits." -ForegroundColor Green
}

Write-Host ""
Write-Host "Starte Claude Code interaktiv zum Einloggen..." -ForegroundColor Cyan
Write-Host "  1. 'Yes, I trust this folder' waehlen" -ForegroundColor White
Write-Host "  2. Login-Link im Browser oeffnen und einloggen" -ForegroundColor White
Write-Host "  3. Danach /exit eingeben" -ForegroundColor White
Write-Host ""

# Interaktiv starten mit Volume
podman run -it --rm `
    --name claude-auth-setup `
    -v "${VolumeName}:/home/claude" `
    $ImageName `
    claude

Write-Host ""
Write-Host "[+] Auth-Setup abgeschlossen!" -ForegroundColor Green
Write-Host "Du kannst jetzt .\run.ps1 nutzen fuer autonome Sessions." -ForegroundColor Green
Write-Host ""
