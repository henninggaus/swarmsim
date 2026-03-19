# SwarmSim Autonomous Claude Code Runner

Podman-Setup für autonome Claude Code Sessions über Nacht.

## Einmalige Einrichtung

```powershell
# 1. Podman installieren (falls noch nicht vorhanden)
winget install RedHat.Podman

# 2. Diesen Ordner nach C:\_repos\swarmsim\.autonomous\ kopieren
#    (oder irgendwo anders hin, Pfade im Skript anpassen)

# 3. Image bauen (dauert ~3 Minuten)
cd C:\_repos\swarmsim\.autonomous
podman build -f Dockerfile -t claude-swarmsim .

# 4. Claude Code Auth sicherstellen
#    Entweder: ANTHROPIC_API_KEY als Umgebungsvariable setzen
#    Oder: "claude setup-token" ausfuehren und Token speichern
```

## Benutzung

```powershell
# Standard-Session (Refactoring + Tests + Doku + Features)
.\run.ps1

# Eigener Prompt
.\run.ps1 -Prompt "Schreibe Unit-Tests fuer alle Packages mit >80% Coverage"

# Prompt aus Datei
.\run.ps1 -PromptFile mein-prompt.md

# Anderes Repo
.\run.ps1 -RepoPath "C:\_repos\anderes-projekt"
```

## Während der Session

```powershell
# Live-Output beobachten
podman logs -f swarmsim-autonomous

# Stoppen
podman stop swarmsim-autonomous

# Status prüfen
podman ps
```

## Am nächsten Morgen

```powershell
cd C:\_repos\swarmsim

# Was hat Claude gemacht?
git log --oneline

# Diff anschauen
git diff main..HEAD

# PROGRESS.md lesen
cat PROGRESS.md

# Wenn alles gut: in main mergen
git checkout main
git merge autonomous/YYYYMMDD-HHMMSS

# Wenn Mist: Branch löschen
git checkout main
git branch -D autonomous/YYYYMMDD-HHMMSS
```

## Auth-Hinweis

Das Skript mountet `~/.claude` read-only in den Container, damit Claude Code
dein bestehendes OAuth-Token (Max 20x Plan) nutzen kann. Alternativ kannst du
`ANTHROPIC_API_KEY` als Umgebungsvariable setzen — dafür im run.ps1 die Zeile
`-e ANTHROPIC_API_KEY=$env:ANTHROPIC_API_KEY` ergänzen.
