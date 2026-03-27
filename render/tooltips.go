package render

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TooltipState tracks hover state for tooltips.
type TooltipState struct {
	HoverID    string // ID of element being hovered
	HoverTicks int    // how many ticks hovering on this element
	Visible    bool   // tooltip is showing
	X, Y       int    // tooltip position
	Lines      []string
}

const tooltipDelay = 25 // ~0.42 seconds at 60fps — fast feedback for discoverability
const tooltipMaxW = 42  // chars per line before wrap

var tooltipRegistry = map[string]string{
	// Tab buttons
	"tab:0": "Arena: Alles rund um die Spielwelt — Hindernisse, Labyrinth, Lichtquelle, Pakete, LKW, Energie.",
	"tab:1": "Evo: Evolution und Lernen — Genetischer Algorithmus, Genetische Programmierung, Neuronale Netze, Teams.",
	"tab:2": "Anzeige: Visualisierungen ein-/ausschalten — Dashboard, Spuren, Heatmap, Minimap und mehr.",
	"tab:3": "Werkzeuge: Geschwindigkeit, Zeitreise, Turnier, Bildschirmfoto, GIF, Daten-Export.",

	// Buttons
	"deploy":     "Laedt das aktuelle Programm auf alle Bots. Alle Bots fuehren danach die Regeln im Editor aus.",
	"reset":      "Setzt alle Bots auf zufaellige Startpositionen zurueck. Statistiken und Fitness werden ebenfalls zurueckgesetzt.",
	"text_mode":  "SwarmScript als Text editieren — jede Zeile ist eine IF...THEN Regel. Volle Kontrolle ueber die Bot-Logik.",
	"block_mode": "Visueller Block-Editor — Regeln per Dropdown zusammenklicken. Ideal fuer Einsteiger, kein Tippen noetig.",
	"bots_plus":  "Mehr Bots hinzufuegen (+10). Mehr Bots = komplexeres emergentes Verhalten, aber hoehere CPU-Last.",
	"bots_minus": "Bots entfernen (-10). Weniger Bots = schnellere Simulation, aber weniger Schwarm-Effekte.",
	"copy":       "Programm in die Zwischenablage kopieren.",
	"paste":      "Programm aus der Zwischenablage einfuegen.",

	// Toggles
	"obstacles": "Hindernisse erscheinen zufaellig in der Arena. Die Bots koennen sehen ob etwas im Weg ist (Sensor 'obs_ahead') und muessen drumherum navigieren. Wie Moebel in einem Raum — der Staubsauger-Roboter muss ausweichen!",
	"maze":      "Ein Labyrinth wird generiert! Bots koennen links und rechts Waende spueren. Der Trick: Immer der rechten Wand folgen fuehrt IMMER zum Ausgang (Rechte-Hand-Regel). Probier das Preset 'Maze Explorer'!",
	"light":     "Eine Lichtquelle erscheint in der Arena. Bots koennen messen wie hell es ist (0=dunkel, 100=direkt am Licht). Damit navigieren sie zum Licht — wie Motten die zur Lampe fliegen!",
	"walls":     "BOUNCE = Bots prallen am Rand ab wie ein Billardball. WRAP = Bots die links rauslaufen kommen rechts wieder rein (wie Pac-Man). WRAP ist interessanter fuer Schwarmverhalten!",
	"delivery":  "Paket-Liefersystem: Es erscheinen farbige Stationen. Gefuellte Kreise = hier Paket abholen (Pickup). Ringe = hier abliefern (Dropoff). Aufgabe: Bots muessen rotes Paket zum roten Ring bringen, blaues zum blauen, etc. Wie ein Logistik-Puzzle!",
	"trucks":    "LKW-Entladung: Ein LKW faehrt an die Rampe, Bots muessen die Pakete rausholen und abliefern. Aber: Max. 3 Bots passen gleichzeitig auf die Rampe — es entsteht ein Stau-Problem! Die Bots muessen sich koordinieren, ohne miteinander zu reden.",
	"evolution": "Evolution AN: Die Zahlenwerte ($A, $B, ...) in deinem Programm werden automatisch verbessert. Wie funktioniert das? Die 20% besten Bots (hoechste Fitness) vererben ihre Werte an die naechste Generation. Zufaellige kleine Aenderungen (Mutation) sorgen fuer Vielfalt. Nach ein paar Generationen siehst du: die Fitness steigt!",
	"gp":        "Genetische Programmierung: Hier evolvieren nicht nur Zahlen, sondern die PROGRAMME selbst! Jeder Bot bekommt eigene Regeln. Die besten Programme werden gemischt (wie zwei Kochrezepte kombinieren) und leicht veraendert. Nach vielen Generationen entstehen Programme, die kein Mensch so geschrieben haette!",
	"teams":     "Wettbewerb! Blaues Team A gegen Rotes Team B — wer liefert mehr Pakete? Beide Teams haben verschiedene Programme und konkurrieren in der gleichen Arena. Wie ein Fussballspiel, nur mit Lieferrobotern!",
	"neuro":     "Neuroevolution: Jeder Bot bekommt ein kleines 'Gehirn' (neuronales Netz). Es nimmt 12 Sensorwerte rein und entscheidet eine von 8 Aktionen — OHNE dass du Regeln schreibst! Die Gehirne der besten Bots werden vererbt und leicht veraendert. Beobachte: Am Anfang laufen die Bots ziellos, nach ein paar Generationen liefern sie gezielt Pakete!",

	// Dropdown presets
	"preset:Aggregation":        "Aggregation: Bots finden sich zu Clustern. Jeder Bot dreht sich zum naechsten Nachbarn (TURN_TO_NEAREST). Simuliert soziale Anziehung — wie Schwarmfische sich zusammenfinden.",
	"preset:Dispersion":         "Dispersion: Bots verteilen sich gleichmaessig im Raum. Wenn ein Nachbar zu nah ist (<40px), dreht sich der Bot weg (TURN_FROM_NEAREST). Gegenteil von Aggregation.",
	"preset:Orbit":              "Orbit: Bots umkreisen die Lichtquelle. Wenn Licht stark (>80), dreht der Bot 90 Grad — erzeugt Kreisbahnen. Benoetigt Light ON. Simuliert Phototaxis bei Insekten.",
	"preset:Color Wave":         "Color Wave: Eine Farbwelle breitet sich durch den Schwarm aus. Ein Bot wird rot, Nachbarn kopieren die Farbe per Nachricht. Zeigt Informationsausbreitung in dezentralen Systemen.",
	"preset:Flocking":           "Flocking (Boids): Schwarmverhalten wie bei Voegeln nach Craig Reynolds. 3 Kraefte: Separation (Abstand halten), Alignment (gleiche Richtung), Cohesion (zusammenbleiben).",
	"preset:Snake Formation":    "Snake: Bots bilden Ketten durch Follow-Mechanik. Jeder Bot folgt dem naechsten (FOLLOW_NEAREST). State-Machine: State 0 = suchen, State 1 = folgen. Emergente Schlangen-Formation.",
	"preset:Obstacle Nav":       "Obstacle Navigation: Bots navigieren um Hindernisse zur Lichtquelle. Kombination aus AVOID_OBSTACLE und TURN_TO_LIGHT. Benoetigt Obstacles ON. Testet reaktive Navigation.",
	"preset:Pulse Sync":         "Pulse Sync: Synchronisierte Lichtblitze wie Gluehwuermchen. Bots haben interne Timer (counter), blitzen gleichzeitig. Erforscht Synchronisation in biologischen Systemen (Kuramoto-Modell).",
	"preset:Trail Follow":       "Trail Follow: Bots kopieren die LED-Farbe des naechsten Nachbarn. Farbspuren breiten sich aus. Zeigt wie lokale Nachahmung zu globalen Mustern fuehrt — emergentes Verhalten.",
	"preset:Ant Colony":         "Ant Colony: Vereinfachter Ameisenalgorithmus. Bots suchen die Lichtquelle (Futter), kommunizieren Fundstellen per SEND_MESSAGE an Nachbarn. Benoetigt Light ON.",
	"preset:Simple Delivery":    "Simple Delivery: Bots explorieren zufaellig, sammeln Pakete an Pickup-Stationen ein (PICKUP) und bringen sie zur passenden Dropoff (GOTO_DROPOFF, DROP). Grundlage fuer alle Delivery-Szenarien.",
	"preset:Delivery Comm":      "Delivery mit Kommunikation: Wie Simple Delivery, aber Bots senden Nachrichten wenn sie Pakete finden (SEND_MESSAGE). Nachbarn koennen reagieren — verbessert die Effizienz.",
	"preset:Delivery Roles":     "Delivery mit Rollen: 50% der Bots sind Scouts (State 0, erkunden) und 50% Carrier (State 1, transportieren). Spezialisierung durch einfache Rollenzuweisung zeigt Effizienzgewinn.",
	"preset:Simple Unload":      "Simple Unload: Grundlegende LKW-Entladung. Bots fahren zur Rampe, nehmen Pakete und bringen sie zur Dropoff. Benoetigt Trucks ON. Ohne Kommunikation — rein reaktiv.",
	"preset:Coordinated Unload": "Coordinated Unload: LKW-Entladung mit LED-Gradient (Wegweiser) und Beacon-Signalen. Bots koordinieren sich fuer effizientere Entladung. Zeigt wie Stigmergie (indirekte Kommunikation) hilft.",
	"preset:Evolving Delivery":  "Evolving Delivery: Delivery-Programm mit evolvierbaren Parametern ($A-$Z). Aktiviere Evolution ON — die Werte optimieren sich ueber Generationen. Beobachte wie Fitness steigt!",
	"preset:Evolving Truck":     "Evolving Truck: Truck-Entladung mit evolvierbaren Parametern. Kombination aus Genetischem Algorithmus und Logistik-Aufgabe. Aktiviere Evolution ON.",
	"preset:Maze Explorer":      "Maze Explorer: Rechte-Hand-Regel (Wall-Following). Bot haelt immer die rechte Wand und findet so jeden Ausgang. Klassischer Algorithmus aus der Robotik. Benoetigt Maze ON.",
	"preset:GP: Random Start":   "GP Random Start: Genetische Programmierung mit komplett zufaelligen Programmen. Jeder Bot startet mit anderen Regeln. Evolution findet die besten Strategien von Null.",
	"preset:GP: Seeded Start":   "GP Seeded Start: GP startet mit Simple Delivery als Basis. Die Evolution verbessert ein bereits funktionierendes Programm weiter. Oft schnellere Konvergenz als Random Start.",
	"preset:Neuro: Delivery":    "Neuroevolution: Jeder Bot hat ein eigenes neuronales Netz (12 Sensoren -> 6 Hidden -> 8 Aktionen = 120 Gewichte). Statt Regeln zu programmieren, lernt das Netz durch Evolution der Gewichte. NEURO wird automatisch aktiviert. Beobachte wie Bots von zufaelligem Verhalten zu gezieltem Liefern lernen!",
	"preset:Neuro: LKW":         "Neuro LKW-Entladung: Neuronales Netz lernt Pakete vom LKW zu entladen und zu sortieren! Truck-spezifische Sensoren (Rampe, LKW-Naehe, Paket-Distanz) und Aktionen (GOTO_RAMP). NEURO+TRUCKS werden automatisch aktiviert. Schwieriger als Delivery — die Bots muessen Rampe finden, Paket greifen UND zur richtigen Station bringen!",

	// Tab 0 (Arena) — new toggles
	"energy":      "Energie-System: Bots verbrauchen Energie bei Bewegung und laden an Stationen auf. Wenn Energie = 0, bleibt der Bot stehen.",
	"dynamicenv":  "Dynamische Umgebung: Hindernisse bewegen sich, Pakete verfallen nach einer Zeit. Erhoehter Schwierigkeitsgrad.",
	"daynight":    "Tag/Nacht-Zyklus: Die Helligkeit aendert sich periodisch. Bots sehen nachts weniger weit. Beeinflusst den 'light' Sensor.",
	"arenaeditor": "Arena-Editor: Klicke in die Arena um Hindernisse (1), Stationen (2) zu platzieren oder zu loeschen (3).",

	// Tab 1 (Evo) — new toggles
	"pareto":      "Pareto: Statt nur EINE Zahl zu optimieren, werden mehrere Ziele gleichzeitig verfolgt (z.B. schnell UND korrekt liefern). Wie im echten Leben: Man kann nicht alles haben — Pareto zeigt die besten Kompromisse.",
	"speciation":  "Speziation: Bots werden in 'Arten' gruppiert (aehnliche Strategien). Kreuzung nur innerhalb einer Art — wie in der Natur. Warum? Damit neue, ungewoehnliche Strategien nicht sofort von den Mainstream-Bots verdraengt werden. Schuetzt Innovation!",
	"sensornoise": "Sensor-Rauschen: Im echten Leben sind Sensoren nie perfekt! Hier liefern sie 15% ungenaue Werte und fallen manchmal aus (2%). Testet ob dein Programm auch mit schlechten Daten klarkommt — genau wie echte Roboter.",
	"memory":      "Bot-Gedaechtnis: Jeder Bot merkt sich welche Zellen er besucht hat. Sensoren 'visited_here' und 'visited_ahead' werden aktiv.",
	"leaderboard": "Leaderboard: Zeigt die Highscore-Tabelle der besten Bots aller Zeiten, sortiert nach Fitness.",

	// Tab 2 (Anzeige) — visualization toggles
	"dashboard":    "Dashboard mit Echtzeit-Statistiken: Fitness-Graph, Delivery-Rate, Heatmap, Bot-Ranking, Event-Ticker.",
	"minimap":      "Minimap: Kleine Uebersichtskarte der gesamten Arena in der Ecke. Zeigt alle Bot-Positionen.",
	"trails":       "Bot-Trails: Zeigt die Bewegungsspuren aller Bots als farbige Linien. Gut um Muster zu erkennen.",
	"heatmap":      "Heatmap: Zeigt welche Bereiche der Arena am meisten besucht werden. Blau = wenig, Rot = viel.",
	"routes":       "Lieferrouten: Zeigt Linien von Pickup zu Dropoff. Bei tragenden Bots: Linie zum Ziel.",
	"livechart":    "Live-Chart: Lieferstatistik als Liniendiagramm in Echtzeit. Zeigt Deliveries und Correct-Rate ueber Zeit.",
	"commgraph":    "Kommunikations-Graph: Zeigt Nachrichten-Linien zwischen Bots die gerade kommunizieren.",
	"msgwaves":     "Nachrichten-Wellen: Zeigt expandierende Ringe wenn Bots SEND_MESSAGE nutzen. Visualisiert Broadcast-Reichweite.",
	"genomeviz":    "Genom-Visualisierung: Zeigt die evolvierten Parameter ($A-$Z) als Balkendiagramm pro Bot.",
	"genomebrowser": "Genom-Browser: Sortierbare Liste aller Bots mit Fitness, Alter und Delivery-Statistik. Tabs: Fitness/Alter/Deliveries.",
	"swarmcenter":  "Schwarm-Mittelpunkt: Zeigt den Massenschwerpunkt aller Bots und den Spread-Radius als Kreis.",
	"congestion":   "Stau-Zonen: Markiert Bereiche mit hoher Bot-Dichte farbig. Hilft Engpaesse zu erkennen.",
	"prediction":   "Vorhersage-Pfeile: Zeigt wohin sich jeder Bot im naechsten Tick bewegen wird. Gut zum Debuggen.",
	"colorfilter":  "Farbfilter: Hebt Bots nach Kriterium hervor — Rot, Gruen, Blau, Traegt Paket, oder Idle. Zyklisch durchklicken.",

	// Algo-Labor (F4) — buttons
	"algo:radar":   "Algorithmus-Radar: Spider-Chart das die Performance aller aktiven Algorithmen auf 4 Achsen vergleicht. Braucht 2+ aktive Algorithmen.",
	"algo:tourney": "Auto-Turnier: Testet alle Optimierungs-Algorithmen automatisch nacheinander auf der aktuellen Fitness-Landschaft und vergleicht die Ergebnisse.",

	// Algo-Labor (F4) — individual algorithms (Aha-Effekt: Natur → Kernidee → Was du SIEHST)
	"algo:0":  "GWO — Wolfsrudel-Jagd: Ein Rudel hat Alpha (bester Bot), Beta, Delta. Die Woelfe kreisen die Beute ein — anfangs weit ausschweifen (Exploration), dann immer enger (Exploitation). DU SIEHST: Die Punkte im Overlay konvergieren langsam zum Optimum, wie Woelfe die Beute einkreisen. Einer der zuverlaessigsten Algorithmen!",
	"algo:1":  "WOA — Buckelwal-Blasennetz: Buckelwale jagen mit einer Spirale aus Luftblasen, die Fische einschliesst. Jeder Wal entscheidet: einkreisen ODER Spirale fliegen (50/50). DU SIEHST: Die Punkte bewegen sich in Spiralen um das Optimum — wie ein Blasennetz, das sich zusammenzieht. Besonders schoen bei glatten Fitnesslandschaften.",
	"algo:2":  "BFO — E.coli-Bakterien: Bakterien navigieren per Tumble-and-Run: geradeaus schwimmen solange es besser wird, zufaellig taumeln wenn nicht. Die Besten vermehren sich, die Schlechtesten sterben. DU SIEHST: Hektisches Zickzack-Muster das sich langsam verdichtet — wie Bakterien die eine Nahrungsquelle finden. Robust bei Rauschen!",
	"algo:3":  "MFO — Motten am Mondlicht: Motten navigieren per Querkompass zum Mond — bei kuenstlichem Licht fliegen sie Spiralen. Jede Motte umkreist eine 'Flamme' (beste Position) in immer engeren Bahnen. DU SIEHST: Schoene Spiralmuster, die Punkte drehen sich ein wie Motten ums Licht. Schnelle Konvergenz!",
	"algo:4":  "Cuckoo — Kuckuck + Levy-Fluege: Der Kuckuck legt Eier in fremde Nester. Schlechte Nester (25%) werden entdeckt und ersetzt. Die Suche nutzt Levy-Fluege: viele kleine Schritte + seltene RIESIGE Spruenge. DU SIEHST: Die meisten Punkte bewegen sich nur wenig, aber ab und zu springt einer quer durch den Raum — das verhindert, in lokalen Optima stecken zu bleiben!",
	"algo:5":  "DE — Differenz-Vektoren: Nimm 3 zufaellige Bots, berechne den Differenz-Vektor zwischen zweien, addiere ihn zum dritten = Mutant. Ist der Mutant besser? Behalten. Sonst verwerfen. DU SIEHST: Gleichmaessige Verkleinerung der Punktwolke — kein klarer Leader, alle verbessern sich parallel. Der demokratischste Algorithmus!",
	"algo:6":  "ABC — Bienenvolk: 3 Rollen: Sammlerinnen suchen lokal um bekannte Quellen, Zuschauer waehlen die besten Quellen (fitness-proportional), Spaeher erkunden komplett neue Gebiete. DU SIEHST: Cluster um gute Stellen mit einzelnen Ausreissern die Neues finden — wie ein echter Bienenstock! Gute Balance zwischen Sicherheit und Abenteuer.",
	"algo:7":  "HSO — Jazz-Improvisation: Stell dir eine Band vor: 95% spielen bekannte Noten (aus dem Gedaechtnis), 30% davon mit leichter Variation (Pitch Adjust), 5% probieren was voellig Neues. DU SIEHST: Die meisten Punkte wandern nur leicht, ab und zu taucht einer ganz woanders auf. Wie Jazz: Struktur + Improvisation!",
	"algo:8":  "Bat — Fledermaus-Echoortung: Fledermaeuse senden Ultraschall und passen Frequenz, Lautstaerke und Pulsrate an. Laut + niedrige Frequenz = weit suchen. Leise + hohe Frequenz = praezise orten. DU SIEHST: Anfangs grosse Spruenge (laute Echos), spaeter Feintuning (leise, praezise). Die Punkte werden mit der Zeit ruhiger.",
	"algo:9":  "HHO — Harris-Bussard-Jagd: Diese Greifvoegel jagen im Team! Phase 1: Umherschweifen und Beute suchen. Phase 2: Beute ist muede (Energie sinkt) — Ueberraschungsangriff! 4 Strategien je nach Fluchtenergie. DU SIEHST: Erst weite Verteilung, dann ploetzlich stuermen alle Punkte zum Optimum — wie ein koordinierter Luftangriff!",
	"algo:10": "SSA — Salpen-Kette im Ozean: Salpen bilden Ketten: der Leader navigiert zur Nahrung, jeder Follower mittelt seine Position mit dem Vordermann. DU SIEHST: Eine Kette von Punkten die sich durchs Overlay zieht — der vordere nah am Optimum, die hinteren folgen nach. Einfach und elegant!",
	"algo:11": "GSA — Newtons Gravitation: Gute Loesungen sind 'schwer' und ziehen schlechtere an — wie Planeten! Die Gravitationskonstante sinkt ueber Zeit: anfangs starke Anziehung (schnelle Bewegung), spaeter nur noch Feintuning. DU SIEHST: Die Punkte 'fallen' aufeinander zu wie ein kollabierendes Sonnensystem. Physik pur!",
	"algo:12": "FPA — Blumen-Bestaeubung: Insekten (80%) fliegen weite Levy-Fluege zwischen Blumen = globale Bestaeubung. Wind (20%) bewegt Pollen nur lokal. DU SIEHST: Mischung aus weiten Spruengen und lokalem Feinjustieren — wie ein Garten in dem Bienen und Wind zusammenarbeiten. Elegant und effizient!",
	"algo:13": "SA — Metall abkuehlen: Heisses Metall hat Atome die wild springen — auch bergauf! Beim Abkuehlen werden sie waehlerischer: nur noch bergab. Am Anfang akzeptiert der Algo auch SCHLECHTERE Loesungen (um lokale Optima zu verlassen). DU SIEHST: Anfangs chaotisches Springen, dann langsames Einpendeln — wie Atome die ihren Platz finden.",
	"algo:14": "AO — Steinadler-Jagd: 4 Phasen: 1) Hoehenflug (weiter Ueberblick), 2) Konturflug (Beute lokalisieren), 3) Langsamer Abstieg (annaehern), 4) Sturzflug (finaler Angriff auf Optimum). DU SIEHST: Die Punkte kreisen erst weit, dann immer enger — wie ein Adler der sein Ziel fixiert hat!",
	"algo:15": "SCA — Sinus/Kosinus-Wellen: Die Position schwingt per sin/cos um das Optimum — wie ein Pendel! Die Amplitude nimmt ab: grosse Schwingungen (Exploration) werden zu kleinen (Exploitation). DU SIEHST: Oszillierende Punkte die immer kleinere Kreise ziehen — mathematisch elegant und ueberraschend effektiv!",
	"algo:16": "DA — Libellen-Schwarm: 5 Kraefte gleichzeitig: Abstand halten, gleiche Richtung, zusammenbleiben (wie Boids!), plus: zum Futter fliegen, vom Feind weg. DU SIEHST: Ein lebendiger Schwarm der sich wie echte Libellen verhalt — mit den gleichen 3 Boids-Regeln die auch Aggregation im Simulator nutzt! Der Algo der dem Simulator am aehnlichsten ist.",
	"algo:17": "TLBO — Klassenzimmer: Der Lehrer (bester Bot) hebt den Durchschnitt an, dann lernen Schueler paarweise voneinander. KEIN EINZIGER Parameter zu tunen — der einzige Algo der einfach funktioniert! DU SIEHST: Gleichmaessiges Zusammenruecken aller Punkte — keine Ausreisser, kein Chaos. Wie eine Klasse die gemeinsam besser wird.",
	"algo:18": "EO — Physikalisches Gleichgewicht: Partikel streben einen Gleichgewichtszustand an (Mittelwert der 4 Besten). Exponentieller Zerfall steuert: anfangs weit weg, spaeter kaum noch Bewegung. DU SIEHST: Schnelle Konvergenz! Die Punkte rasen anfangs zum Zentrum und pendeln sich dann ein. Gut fuer Probleme wo schnelle Antworten zaehlen.",
	"algo:19": "Jaya — Der Einfachste: Sanskrit fuer 'Sieg'. Regel: Bewege dich zum Besten hin UND vom Schlechtesten weg. Das wars! NULL Parameter, trotzdem ueberraschend gut. DU SIEHST: Alle Punkte wandern gleichmaessig in eine Richtung — weg vom Schlechtesten, hin zum Besten. Proof dass Einfachheit siegt!",

	// Tab 3 (Tools/Werkzeuge)
	"newround":   "Startet eine neue Runde — setzt Team-Scores und Delivery-Statistiken zurueck. Bei Trucks: neuer LKW faehrt vor.",
	"replay":     "Replay-Modus: Pausiert die Simulation und erlaubt Zeitreisen durch aufgezeichnete Snapshots.",
	"tournament": "Turnier-Modus: Verschiedene Programme treten in der gleichen Arena gegeneinander an.",
	"screenshot": "Speichert einen Screenshot der aktuellen Ansicht als PNG-Datei.",
	"gif":        "GIF-Aufnahme: Startet/stoppt die Aufnahme eines animierten GIF der Simulation.",
	"exportcsv":  "Exportiert die aktuellen Statistiken als CSV-Daten in die Zwischenablage.",
	"exportswarm": "Speichert den gesamten Zustand (Programm, Einstellungen, Bot-Positionen) als .swarm Datei. Ideal zum Teilen von Experimenten.",
	"importswarm": "Laedt einen gespeicherten Zustand aus einer .swarm Datei. Stellt Programm, Einstellungen und Arena wieder her.",
	"speed:0.5": "Halbe Geschwindigkeit — ideal zum Beobachten einzelner Bot-Entscheidungen und Debugging.",
	"speed:1":   "Normale Geschwindigkeit (1x) — Echtzeit-Simulation.",
	"speed:2":   "Doppelte Geschwindigkeit — schnellere Ergebnisse, gut fuer kurze Experimente.",
	"speed:5":   "5-fache Geschwindigkeit — fuer laengere Evolutionslaeufe. Details schwerer erkennbar.",
	"speed:10":  "10-fache Geschwindigkeit — Turbo-Modus fuer Evolution und Algorithmen-Benchmarks. Hohe CPU-Last.",

	// Stats panel items
	"stat:chains":       "Chains = Anzahl der Bot-Ketten (Follow-Formationen). Bots folgen dem naechsten Nachbarn mit FOLLOW_NEAREST. Laengere Ketten zeigen starke soziale Bindung.",
	"stat:coverage":     "Coverage = Prozent der Arena, die von Bots abgedeckt wird. 100% = Bots verteilt in alle Bereiche. Niedrig = Bots clustern zusammen. 20x20 Raster-Berechnung.",
	"stat:neighbors":    "Avg Neighbors = Durchschnittliche Nachbar-Anzahl pro Bot (120px Radius). Hoch = dicht gepackter Schwarm. Niedrig = Bots verteilt. Beeinflusst Kommunikations-Effizienz.",
	"stat:delivered":    "Geliefert = Anzahl erfolgreich zugestellter Pakete. Richtig = zur passenden Farbe geliefert. Rate = Korrektheit in Prozent. Ziel: 100% Korrektheit!",
	"stat:carrying":     "Tragen = Bots die gerade ein Paket transportieren. Suchen = Bots ohne Paket. Ideale Balance haengt vom Programm ab.",
	"stat:avgtime":      "Durchschn. Lieferzeit = Ticks von Pickup bis Dropoff gemittelt. Niedrigere Werte = effizientere Lieferstrategie. Optimiere mit Evolution!",
}

// UpdateTooltip updates tooltip state based on current hover position.
func UpdateTooltip(ts *TooltipState, mx, my int, currentHitID string) {
	if currentHitID == "" || currentHitID == "editor" || currentHitID == "botcount" {
		ts.HoverID = ""
		ts.HoverTicks = 0
		ts.Visible = false
		return
	}

	if currentHitID == ts.HoverID {
		ts.HoverTicks++
	} else {
		ts.HoverID = currentHitID
		ts.HoverTicks = 0
		ts.Visible = false
	}

	if ts.HoverTicks >= tooltipDelay {
		text, ok := tooltipRegistry[currentHitID]
		if ok {
			ts.Visible = true
			ts.Lines = wrapTooltipText(text, tooltipMaxW)
			ts.X = mx + 12
			ts.Y = my - len(ts.Lines)*lineH - 8
			// Clamp to screen
			tipW := tooltipPixelWidth(ts.Lines)
			if ts.X+tipW > 1270 {
				ts.X = 1270 - tipW
			}
			if ts.Y < 5 {
				ts.Y = my + 20
			}
		}
	}
}

// DrawTooltip renders the tooltip if visible.
func DrawTooltip(screen *ebiten.Image, ts *TooltipState) {
	if !ts.Visible || len(ts.Lines) == 0 {
		return
	}

	w := tooltipPixelWidth(ts.Lines) + 12
	h := len(ts.Lines)*lineH + 8

	// Background
	vector.DrawFilledRect(screen, float32(ts.X-2), float32(ts.Y-2), float32(w+4), float32(h+4), color.RGBA{0, 0, 0, 230}, false)
	vector.StrokeRect(screen, float32(ts.X-2), float32(ts.Y-2), float32(w+4), float32(h+4), 1, color.RGBA{100, 120, 160, 200}, false)

	// Text
	for i, line := range ts.Lines {
		printColoredAt(screen, line, ts.X+4, ts.Y+4+i*lineH, color.RGBA{220, 225, 240, 255})
	}
}

func wrapTooltipText(text string, maxChars int) []string {
	words := strings.Fields(text)
	var lines []string
	var current string
	for _, word := range words {
		if current == "" {
			current = word
		} else if len(current)+1+len(word) <= maxChars {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func tooltipPixelWidth(lines []string) int {
	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	return maxLen * charW
}
