# SwarmSim — Schwarm-Robotik-Simulator

Ein 2D-Schwarm-Robotik-Simulator mit eigener Skriptsprache (SwarmScript), genetischem Algorithmus und dezentraler Multi-Agent-Logistik. Gebaut in Go mit [Ebiten](https://ebitengine.org/).

## Features

- **7 Simulationsmodi** (F1-F7): Foraging, Labyrinth, Energy Crisis, Sandbox, Evolution, LKW-Entladung, Swarm-Editor
- **SwarmScript**: Eigene DSL zum Programmieren von Bot-Verhalten mit 30+ Sensoren und 25+ Aktionen
- **Dezentrales Delivery-System**: Farbcodierte Pickup/Dropoff-Stationen mit emergenter LED-Gradient-Navigation
- **Genetischer Algorithmus**: 7-Gen-Genom mit Crossover, Mutation und Fitness-Tracking
- **Pheromon-System** (Ant Colony Optimization): Search, Found-Resource und Danger Pheromone
- **LKW-Entlade-Szenario**: Kooperatives Heben, Sortierzonen, Timer-basierte Bewertung
- **Minimap**, **Screenshot** (PNG) und **GIF-Recording**
- **WebAssembly-Build** zum Spielen im Browser

## Screenshots

![Delivery Mode](screenshots/delivery.png)
![Snake Formation](screenshots/snake.png)
![Truck Mode](screenshots/truck.png)
![SwarmScript Editor](screenshots/editor.png)

## Installation

```bash
# Clone
git clone https://github.com/henning-heisenberg/swarmsim.git
cd swarmsim

# Build
make build          # Linux/Mac
make windows        # Windows .exe
make wasm           # WebAssembly

# Oder direkt
go build -o swarmsim .

# Run
./swarmsim
```

### Voraussetzungen

- Go 1.21+ (getestet mit Go 1.25)
- Keine CGO-Abhängigkeiten

## Szenarien

| Taste | Szenario | Beschreibung |
|-------|----------|--------------|
| **F1** | Foraging Paradise | 2000x1500 Arena, 50 respawnende Ressourcen, Fokus auf Schwarm-Sammeln |
| **F2** | Labyrinth | Labyrinth-Generator mit Sackgassen-Ressourcen, Tanks räumen Hindernisse |
| **F3** | Energy Crisis | 2x Energieverbrauch, langsamer Respawn, Healer-Strategie entscheidend |
| **F4** | Sandbox | Standardkonfiguration, freies Experimentieren |
| **F5** | Evolution Arena | Schnelle Generationen (500 Ticks), automatische Evolution |
| **F6** | LKW-Entladung | Truck-Modus mit Paketen, Sortierzonen und Timer |
| **F7** | Programmable Swarm | SwarmScript-Editor mit 800x800 Arena und bis zu 500 Bots |

## Bot-Typen

| Typ | Farbe | Speed | Sensor | Comm | Spezial |
|-----|-------|-------|--------|------|---------|
| **Scout** | Cyan | 3.0 | 150px | 80px | Erkundet, markiert Ressourcen, Search-Pheromon |
| **Worker** | Orange | 1.5 | 60px | 60px | Sammelt und transportiert, folgt Found-Pheromon |
| **Leader** | Gold | 1.0 | 100px | 200px | Koordiniert Workers, relayed Nachrichten |
| **Tank** | Dunkelgrün | 0.8 | 50px | 50px | Schiebt Hindernisse, reagiert auf Hilfe-Anfragen |
| **Healer** | Pink | 1.2 | 80px | 80px | Heilt Bots und lädt Energie auf |

## Tastatur-Shortcuts

### Global (alle Modi)

| Taste | Aktion |
|-------|--------|
| **Space** | Pause / Fortsetzen |
| **+/-** | Simulationsgeschwindigkeit |
| **F1-F5** | Szenarien laden |
| **F6** | LKW-Szenario |
| **F7** | Swarm-Editor |
| **F10** | Screenshot (PNG) |
| **F11** | GIF-Recording Start/Stop |
| **F12** | CPU-Profiling (Build-Tag `profile`) |
| **ESC** | Beenden |

### Standard-Modus (F1-F6)

| Taste | Aktion |
|-------|--------|
| **Linksklick** | Bot auswählen |
| **Rechtsklick + Ziehen** | Kamera bewegen |
| **Mausrad** | Zoom |
| **WASD** | Kamera bewegen |
| **1-5** | Scout/Worker/Leader/Tank/Healer spawnen |
| **R** | Ressource spawnen |
| **H** | Hindernis platzieren |
| **F** | Kommunikations-Radius anzeigen |
| **G** | Sensor-Radius anzeigen |
| **D** | Debug-Kommunikationslinien |
| **P** | Pheromon-Visualisierung (OFF/FOUND/ALL) |
| **E** | Generation erzwingen (Evolution) |
| **V** | Genom-Overlay für ausgewählten Bot |
| **T** | Trail-Rendering |
| **M** | Minimap (nur bei Zoom > 1.0) |
| **N** | Neuer LKW (nur Truck-Modus) |

### Swarm-Modus (F7)

| Taste | Aktion |
|-------|--------|
| **Linksklick** | Bot/UI-Element auswählen |
| **T** | Trails anzeigen |
| **C** | Delivery-Routen anzeigen |
| **L** | Lichtquelle setzen/entfernen |
| **M** | Minimap |

## SwarmScript

SwarmScript ist eine regelbasierte DSL zum Programmieren von Bot-Verhalten. Jeder Bot evaluiert die Regeln von oben nach unten; die erste passende Regel wird ausgeführt.

### Syntax

```
# Kommentar
IF <bedingung> [AND <bedingung>...] THEN <aktion>
IF true THEN <aktion>                               # Default-Regel
```

### Beispiel: Smart Delivery

```
# Paket aufheben wenn nahe an Pickup mit Paket
IF carrying == 0 AND pickup_dist < 20 AND has_pkg == 1 THEN PICKUP

# Mit Paket zum passenden Dropoff navigieren
IF carrying == 1 AND match == 1 THEN GOTO_MATCH
IF carrying == 1 THEN GOTO_LED

# LED-Farbe für Navigation setzen
IF carrying == 1 THEN LED_DROPOFF
IF carrying == 0 THEN LED_PICKUP

# Paket abliefern
IF carrying == 1 AND dropoff_dist < 20 AND match == 1 THEN DROP

# Standardverhalten
IF obs_ahead == 1 THEN AVOID_OBSTACLE
IF near_dist < 15 THEN TURN_FROM_NEAREST
IF true THEN FWD
```

### Sensor-Referenz

| Sensor | Alias | Beschreibung |
|--------|-------|--------------|
| `neighbors_count` | `nbrs`, `neighbors` | Anzahl Nachbarn im Sensorbereich |
| `nearest_distance` | `near_dist` | Distanz zum nächsten Nachbarn |
| `state` | `my_state` | Interner Zustand (0-255) |
| `counter` | — | Zähler-Variable (0-255) |
| `timer` | — | Timer (zählt runter) |
| `on_edge` | `edge` | Am Arena-Rand? (true/false) |
| `received_message` | `msg` | Empfangene Nachricht |
| `light_value` | `light` | Lichtstärke (0-100) |
| `random` | `rnd` | Zufallswert (0-100) |
| `has_leader` | `leader` | Folgt einem Bot? |
| `has_follower` | `follower` | Wird gefolgt? |
| `chain_length` | `chain_len` | Länge der Follow-Kette |
| `nearest_led_r/g/b` | — | LED-Farbe des nächsten Nachbarn |
| `obstacle_ahead` | `obs_ahead` | Hindernis voraus? |
| `obstacle_distance` | `obs_dist` | Distanz zum Hindernis |
| `value1`, `value2` | — | Benutzerdefinierte Variablen |
| `tick` | — | Simulations-Tick |

**Delivery-Sensoren:**

| Sensor | Alias | Beschreibung |
|--------|-------|--------------|
| `carrying` | `carry` | Trägt Paket? (0/1) |
| `carrying_color` | — | Farbe des Pakets (1-4) |
| `nearest_pickup_dist` | `p_dist` | Distanz zur nächsten Pickup-Station |
| `nearest_pickup_color` | `pickup_color` | Farbe der nächsten Pickup-Station |
| `nearest_pickup_has_package` | `has_pkg` | Hat Pickup ein Paket? |
| `nearest_dropoff_dist` | `d_dist` | Distanz zur nächsten Dropoff-Station |
| `dropoff_match` | `match` | Passt Dropoff zur Paketfarbe? |
| `nearest_matching_led_dist` | `led_dist` | Distanz zum Bot mit passender LED |
| `heard_pickup_color` | `heard_pickup` | Per Nachricht gehörte Pickup-Farbe |
| `heard_dropoff_color` | `heard_dropoff` | Per Nachricht gehörte Dropoff-Farbe |

### Aktions-Referenz

| Aktion | Alias | Beschreibung |
|--------|-------|--------------|
| `MOVE_FORWARD` | `FWD` | Vorwärts bewegen |
| `MOVE_FORWARD_SLOW` | `FWD_SLOW` | Langsam vorwärts |
| `STOP` | — | Anhalten |
| `TURN_LEFT N` | — | Links drehen (N Grad) |
| `TURN_RIGHT N` | — | Rechts drehen (N Grad) |
| `TURN_RANDOM` | — | Zufällige Richtung |
| `TURN_TO_NEAREST` | — | Zum nächsten Nachbarn drehen |
| `TURN_FROM_NEAREST` | — | Vom nächsten Nachbarn weg |
| `TURN_TO_CENTER` | — | Zum Zentrum der Nachbarn |
| `TURN_TO_LIGHT` | — | Zur Lichtquelle |
| `TURN_AWAY_OBSTACLE` | `AVOID_OBSTACLE` | Hindernis ausweichen |
| `FOLLOW_NEAREST` | — | Nächstem Bot folgen |
| `UNFOLLOW` | — | Aufhören zu folgen |
| `SET_STATE N` | — | Zustand setzen |
| `SET_COUNTER N` | — | Zähler setzen |
| `INC_COUNTER` | — | Zähler +1 |
| `DEC_COUNTER` | — | Zähler -1 |
| `SET_VALUE1 N` | — | Value1 setzen |
| `SET_VALUE2 N` | — | Value2 setzen |
| `SET_TIMER N` | — | Timer setzen (Ticks) |
| `SEND_MESSAGE N` | — | Nachricht senden |
| `SET_LED R G B` | — | LED-Farbe setzen (RGB) |
| `COPY_NEAREST_LED` | `COPY_LED` | LED des Nachbarn kopieren |
| `PICKUP` | — | Paket aufheben |
| `DROP` | — | Paket ablegen |
| `TURN_TO_PICKUP` | `GOTO_PICKUP` | Zur Pickup-Station |
| `TURN_TO_MATCHING_DROPOFF` | `GOTO_MATCH` | Zum passenden Dropoff |
| `TURN_TO_MATCHING_LED` | `GOTO_LED` | Zum Bot mit passender LED |
| `SEND_PICKUP N` | — | Pickup-Farbe broadcasten |
| `SEND_DROPOFF N` | — | Dropoff-Farbe broadcasten |
| `SET_LED_PICKUP_COLOR` | `LED_PICKUP` | LED = nächste Pickup-Farbe |
| `SET_LED_DROPOFF_COLOR` | `LED_DROPOFF` | LED = nächste Dropoff-Farbe |

### Preset-Programme (13 Stück)

| Name | Beschreibung |
|------|--------------|
| Aggregation | Bots clustern sich zum Zentrum |
| Dispersion | Bots verteilen sich gleichmäßig |
| Orbit | Kreisen um Lichtquelle |
| Color Wave | Rote LED-Welle per Messaging |
| Flocking | Boids-artiges Schwarmverhalten |
| Snake Formation | Bots bilden Ketten und schlängeln |
| Obstacle Nav | Navigation um Hindernisse zum Licht |
| Pulse Sync | Synchronisierte LED-Pulse wie Glühwürmchen |
| Trail Follow | Bots folgen und kopieren LED-Farben |
| Ant Colony | Ameisenartiges Sammeln mit Lichtsuche |
| Simple Delivery | Smart Delivery mit LED-Gradient-Navigation |
| Delivery Comm | Delivery mit Kommunikations-Nachrichten |
| Delivery Roles | Zwei Rollen: Beacon (LED-Leuchtturm) und Carrier |

## Systeme

### Pheromon-System (ACO)
Drei Pheromontypen: **Search** (blau), **Found Resource** (grün), **Danger** (rot). Bots hinterlassen Pheromone, die über Zeit verdampfen und zu Nachbarzellen diffundieren. Scouts meiden Search-Pheromon (um neue Gebiete zu erkunden), Workers folgen Found-Resource-Trails. Visualisierung mit **P**.

### Energie-System
Alle Bots haben Energie (0-100). Bewegung, Messaging, Tragen, Pheromone und Hindernisse schieben verbrauchen Energie. Aufladen an der Home Base. Healers können Energie übertragen. Bei 0 Energie: Bot wird immobilisiert und sendet Hilferuf.

### Genetischer Algorithmus
7 Gene pro Bot: FlockingWeight, PheromoneFollow, ExplorationDrive, CommFrequency, EnergyConservation, SpeedPreference, CooperationBias. Top 30% Genome werden bewahrt, Rest durch Crossover mit Mutation ersetzt. **E** zum manuellen Evolvieren, **V** für Genom-Overlay.

### Delivery-System (Swarm-Modus)
Farbcodierte Pickup/Dropoff-Stationen (Rot, Blau, Gelb, Grün). Bots müssen Pakete von Pickups aufheben und zu passenden Dropoffs liefern. Emergente Navigation über LED-Gradienten: Bots setzen ihre LED-Farbe basierend auf Station-Nähe, andere Bots navigieren entlang des Farbgradienten.

## Architektur

```
swarmsim/
  domain/          Kernlogik (reine Typen, kein Framework)
    bot/           Bot-Interface, Typen (Scout/Worker/Leader/Tank/Healer)
    physics/       Arena, Obstacles, SpatialHash (O(1) Neighbor-Lookup)
    comm/          Dezentrales Messaging (TTL, Range-basiert)
    genetics/      Genom, Crossover, Mutation, Fitness
    resource/      Ressourcen-Spawning und -Management
    swarm/         SwarmBot, SwarmState, Delivery-Stations
  engine/          Orchestrierung
    simulation/    Simulation-Loop, Szenarien, Config
    swarmscript/   Parser + Interpreter für SwarmScript DSL
    pheromone/     Pheromon-Grid mit Diffusion und Evaporation
  render/          Ebiten-basierte Visualisierung
    renderer.go    Kamera, Bot-Sprites, Pheromon-Rendering
    hud.go         Heads-Up Display
    swarm_render.go  Swarm-Modus Arena-Rendering
    swarm_editor.go  Code-Editor mit Syntax-Highlighting
    minimap.go     150x100px Übersichtskarte
    capture.go     Screenshot (PNG) und GIF-Recording
    particles.go   Partikel-Effekte
    colors.go      Farbkonstanten
  main.go          Ebiten Game-Loop, Input-Handling
  profiling.go     CPU-Profiling (Build-Tag: profile)
```

### Performance-Optimierungen
- **SpatialHash**: Pre-allokierte Flat-Slices statt Maps, wiederverwendbarer Query-Buffer
- **Bot-Sprites**: Vorgerenderte 24x24px Dreiecke, Tinting via ColorScale
- **Pheromon-Cache**: Pixel-Buffer nur alle 5 Ticks neu berechnet
- **Text-Cache**: HUD-Texte als gecachte GPU-Images mit 120-Frame-Eviction
- **Bedingte Dashed Lines**: Nur gezeichnet wenn Bots Pakete tragen

## Technologie

- **Go 1.21+** (kein CGO)
- **Ebiten v2.9** (2D Game Library)
- Cross-Compile: Windows, Linux, WebAssembly
- Keine externen Abhängigkeiten außer Ebiten

## Lizenz

MIT
