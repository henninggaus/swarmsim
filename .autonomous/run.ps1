<# 
    SwarmSim Autonomous Claude Code Runner
    
    Verwendung:
      .\run.ps1                          # Standard-Prompt (Refactoring + Tests + Doku)
      .\run.ps1 -Prompt "Dein Prompt"    # Eigener Prompt
      .\run.ps1 -PromptFile prompt.md    # Prompt aus Datei

    Voraussetzungen:
      - Podman installiert und im PATH
      - Einmalig: Auth einrichten mit setup-auth.ps1
#>

param(
    [string]$Prompt = "",
    [string]$PromptFile = "",
    [string]$RepoPath = "C:\_repos\swarmsim",
    [string]$ImageName = "claude-swarmsim",
    [string]$ContainerName = "swarmsim-autonomous",
    [string]$VolumeName = "claude-auth"
)

# --- Farben ---
function Write-Step($msg) { Write-Host "[>] $msg" -ForegroundColor Cyan }
function Write-OK($msg)   { Write-Host "[+] $msg" -ForegroundColor Green }
function Write-Err($msg)  { Write-Host "[!] $msg" -ForegroundColor Red }

# --- Prüfe Voraussetzungen ---
Write-Step "Pruefe Voraussetzungen..."

if (-not (Get-Command podman -ErrorAction SilentlyContinue)) {
    Write-Err "Podman nicht gefunden. Bitte installieren."
    exit 1
}

if (-not (Test-Path $RepoPath)) {
    Write-Err "Repo-Pfad nicht gefunden: $RepoPath"
    exit 1
}

if (-not (Test-Path "$RepoPath\.git")) {
    Write-Err "Kein Git-Repo in $RepoPath"
    exit 1
}

# --- Prüfe ob Auth-Volume existiert ---
$volumeExists = podman volume inspect $VolumeName 2>$null
if (-not $volumeExists) {
    Write-Err "Auth-Volume '$VolumeName' nicht gefunden!"
    Write-Host "Fuehre zuerst .\setup-auth.ps1 aus um dich einzuloggen." -ForegroundColor Yellow
    exit 1
}

# --- Sicherheitsnetz: Branch erstellen ---
Write-Step "Erstelle Sicherheits-Branch..."
Push-Location $RepoPath

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$branchName = "autonomous/$timestamp"

# Aktuellen Stand committen falls uncommitted changes
$status = git status --porcelain
if ($status) {
    Write-Step "Uncommitted changes gefunden, committe erst..."
    git add -A
    git commit -m "chore: save state before autonomous session $timestamp"
}

# Branch erstellen
$currentBranch = git branch --show-current
git checkout -b $branchName
Write-OK "Branch erstellt: $branchName (vorher: $currentBranch)"

Pop-Location

# --- Image bauen falls nötig ---
$imageExists = podman images -q $ImageName 2>$null
if (-not $imageExists) {
    Write-Step "Baue Container-Image..."
    $dockerfilePath = Split-Path -Parent $MyInvocation.MyCommand.Path
    podman build -f "$dockerfilePath\Dockerfile" -t $ImageName "$dockerfilePath"
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Image-Build fehlgeschlagen!"
        exit 1
    }
    Write-OK "Image gebaut: $ImageName"
} else {
    Write-OK "Image existiert bereits: $ImageName"
}

# --- Prompt zusammenbauen ---
if ($PromptFile -and (Test-Path $PromptFile)) {
    $finalPrompt = Get-Content $PromptFile -Raw
    Write-OK "Prompt aus Datei geladen: $PromptFile"
} elseif ($Prompt) {
    $finalPrompt = $Prompt
} else {
    # Standard-Prompt fuer SwarmSim
    $finalPrompt = @"
Du bist ein erfahrener Go-Entwickler und arbeitest autonom am SwarmSim-Projekt in /workspace.

REGELN:
- Arbeite vollstaendig autonom, stelle KEINE Rueckfragen
- Committe nach jedem logischen Schritt mit conventional commits (feat:, fix:, refactor:, test:, docs:)
- Fuehre nach jeder Aenderung "go build ./..." und "go vet ./..." aus
- Wenn Tests existieren, fuehre "go test ./..." nach jeder Aenderung aus
- Wenn ein Ansatz nicht funktioniert, versuche einen anderen
- Hoere NICHT vorzeitig auf. Arbeite systematisch die gesamte Liste ab
- Speichere deinen Fortschritt nach jedem abgeschlossenen Schritt in PROGRESS.md
- Dein Context Window wird automatisch kompaktiert. Lies PROGRESS.md nach jedem Compact um zu wissen wo du warst

AUFGABEN (in dieser Reihenfolge):

Phase 1 - Analyse:
1. Lies die gesamte Codebase und erstelle PROGRESS.md mit einer Uebersicht der Architektur
2. Identifiziere alle Probleme und Verbesserungspotenziale

Phase 2 - Code-Qualitaet:
3. Extrahiere Magic Numbers in benannte Konstanten
4. Pruefe und verbessere Error Handling (kein ignoriertes error return)
5. Entferne Code-Duplizierung, extrahiere gemeinsame Funktionen
6. Stelle sicher dass alle exportierten Typen und Funktionen GoDoc-Kommentare haben

Phase 3 - Tests:
7. Schreibe Unit-Tests fuer alle Packages, Ziel: >70% Coverage
8. Fuehre "go test -cover ./..." aus und dokumentiere die Coverage in PROGRESS.md

Phase 4 - Features:
9. Pruefe ob die Simulation-Parameter sinnvoll konfigurierbar sind
10. Verbessere Logging und Observability wo sinnvoll
11. Pruefe die Package-Struktur und refactore wenn noetig

Phase 5 - Dokumentation:
12. Aktualisiere oder erstelle README.md mit Architektur-Uebersicht, Build-Anleitung, Usage
13. Erstelle ARCHITECTURE.md mit detaillierter Beschreibung der Komponenten

Aktualisiere PROGRESS.md nach JEDEM abgeschlossenen Schritt mit Status und Erkenntnissen.
"@
}

Write-OK "Prompt-Laenge: $($finalPrompt.Length) Zeichen"

# --- Prompt als Datei speichern (vermeidet Shell-Escaping-Probleme) ---
$promptFile = Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "_prompt.tmp.md"
[System.IO.File]::WriteAllText($promptFile, $finalPrompt)
Write-OK "Prompt geschrieben: $promptFile"

# --- Alten Container entfernen falls vorhanden ---
podman rm -f $ContainerName 2>$null | Out-Null

# --- Container starten ---
Write-Step "Starte autonome Claude Code Session..."
Write-Host ""
Write-Host "=============================================" -ForegroundColor Yellow
Write-Host "  Branch:    $branchName" -ForegroundColor Yellow
Write-Host "  Repo:      $RepoPath" -ForegroundColor Yellow  
Write-Host "  Container: $ContainerName" -ForegroundColor Yellow
Write-Host "  Volume:    $VolumeName" -ForegroundColor Yellow
Write-Host "=============================================" -ForegroundColor Yellow
Write-Host ""
Write-Host "Beobachten:   podman logs -f $ContainerName" -ForegroundColor DarkGray
Write-Host "Stoppen:      podman stop $ContainerName" -ForegroundColor DarkGray
Write-Host "Morgen:       cd $RepoPath && git log --oneline" -ForegroundColor DarkGray
Write-Host ""

# Container starten: Prompt wird als Datei gemountet und per cat reingepipet
podman run -d `
    --name $ContainerName `
    -v "${RepoPath}:/workspace" `
    -v "${VolumeName}:/home/claude" `
    -v "${promptFile}:/tmp/prompt.md:ro" `
    $ImageName `
    bash -c "cat /tmp/prompt.md | claude -p --dangerously-skip-permissions"

if ($LASTEXITCODE -eq 0) {
    Write-OK "Container laeuft! Claude arbeitet jetzt autonom."
    Write-Host ""
    Write-Host "Gute Nacht! Morgen: git log --oneline auf Branch '$branchName'" -ForegroundColor Green
} else {
    Write-Err "Container-Start fehlgeschlagen!"
    Write-Host "Debug: podman logs $ContainerName" -ForegroundColor DarkGray
    exit 1
}
