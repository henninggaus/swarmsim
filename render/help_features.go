package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"swarmsim/locale"
)

// drawHelpRightColumnAndMath draws the right column (glossary, features, tips),
// the full-width math section, concepts section, and footer.
// leftEndY is the final Y from the left column, used to compute the vertical separator and math start.
func drawHelpRightColumnAndMath(screen *ebiten.Image, px, midX, y int, scrollY int, leftEndY int) {
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()

	ry := y

	// ==============================================
	// GLOSSAR: Begriffe in Alltagssprache
	// ==============================================
	printColoredAt(screen, locale.T("help.glossary.title"), midX, ry, colorHelpSection)
	ry += lineH + 4

	glossarCol := color.RGBA{255, 220, 140, 255} // warm gold for terms
	glossarDesc := color.RGBA{190, 195, 210, 255}

	// Each term: bold name + plain-language explanation
	glossarItems := []struct{ term, desc1, desc2 string }{
		{locale.T("help.glossary.bot.name"),
			locale.T("help.glossary.bot.desc1"),
			locale.T("help.glossary.bot.desc2")},
		{locale.T("help.glossary.sensor.name"),
			locale.T("help.glossary.sensor.desc1"),
			locale.T("help.glossary.sensor.desc2")},
		{locale.T("help.glossary.fitness.name"),
			locale.T("help.glossary.fitness.desc1"),
			locale.T("help.glossary.fitness.desc2")},
		{locale.T("help.glossary.evolution.name"),
			locale.T("help.glossary.evolution.desc1"),
			locale.T("help.glossary.evolution.desc2")},
		{locale.T("help.glossary.parameter.name"),
			locale.T("help.glossary.parameter.desc1"),
			locale.T("help.glossary.parameter.desc2")},
		{locale.T("help.glossary.emergenz.name"),
			locale.T("help.glossary.emergenz.desc1"),
			locale.T("help.glossary.emergenz.desc2")},
		{locale.T("help.glossary.exploration.name"),
			locale.T("help.glossary.exploration.desc1"),
			locale.T("help.glossary.exploration.desc2")},
		{locale.T("help.glossary.exploitation.name"),
			locale.T("help.glossary.exploitation.desc1"),
			locale.T("help.glossary.exploitation.desc2")},
		{locale.T("help.glossary.konvergenz.name"),
			locale.T("help.glossary.konvergenz.desc1"),
			locale.T("help.glossary.konvergenz.desc2")},
		{locale.T("help.glossary.lokalesoptimum.name"),
			locale.T("help.glossary.lokalesoptimum.desc1"),
			locale.T("help.glossary.lokalesoptimum.desc2")},
	}

	for _, g := range glossarItems {
		printColoredAt(screen, g.term, midX+5, ry, glossarCol)
		ry += lineH
		printColoredAt(screen, "  "+g.desc1, midX+5, ry, glossarDesc)
		ry += lineH
		printColoredAt(screen, "  "+g.desc2, midX+5, ry, glossarDesc)
		ry += lineH + 3
	}

	ry += 4
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 8

	printColoredAt(screen, locale.T("help.features.title"), midX, ry, colorHelpSection)
	ry += lineH + 4

	// --- Emergentes Verhalten ---
	printColoredAt(screen, locale.T("help.emergence.title"), midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.emergence.1"),
		locale.T("help.emergence.2"),
		locale.T("help.emergence.3"),
		locale.T("help.emergence.4"),
		locale.T("help.emergence.5"),
		locale.T("help.emergence.6"),
		locale.T("help.emergence.7"),
	})
	ry += 6

	// --- Delivery ---
	printColoredAt(screen, locale.T("help.delivery.title"), midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.delivery.1"),
		locale.T("help.delivery.2"),
		locale.T("help.delivery.3"),
		locale.T("help.delivery.4"),
	})
	ry += 6

	// --- Trucks ---
	printColoredAt(screen, locale.T("help.trucks.title"), midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.trucks.1"),
		locale.T("help.trucks.2"),
		locale.T("help.trucks.3"),
	})
	ry += 6

	// --- Evolution ---
	printColoredAt(screen, locale.T("help.evolution.title"), midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.evolution.1"),
		locale.T("help.evolution.2"),
		locale.T("help.evolution.3"),
	})
	ry += 6

	// --- GP ---
	printColoredAt(screen, locale.T("help.gp.title"), midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.gp.1"),
		locale.T("help.gp.2"),
		locale.T("help.gp.3"),
	})
	ry += 6

	// --- Neuroevolution ---
	printColoredAt(screen, locale.T("help.neuro.title"), midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.neuro.1"),
		locale.T("help.neuro.2"),
		locale.T("help.neuro.3"),
		locale.T("help.neuro.4"),
	})
	ry += 6

	// --- Algorithmen ---
	printColoredAt(screen, locale.T("help.algos.title"), midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.algos.1"),
		locale.T("help.algos.2"),
		locale.T("help.algos.3"),
		locale.T("help.algos.4"),
		locale.T("help.algos.5"),
		locale.T("help.algos.6"),
		locale.T("help.algos.7"),
	})
	ry += 6

	// --- Teams ---
	printColoredAt(screen, locale.T("help.teams.title"), midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.teams.1"),
		locale.T("help.teams.2"),
		locale.T("help.teams.3"),
	})
	ry += 6

	// --- Kommunikation ---
	printColoredAt(screen, locale.T("help.comm.title"), midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.comm.1"),
		locale.T("help.comm.2"),
		locale.T("help.comm.3"),
	})
	ry += 6

	// --- Dashboard ---
	printColoredAt(screen, locale.T("help.dashboard.title"), midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.dashboard.1"),
		locale.T("help.dashboard.2"),
	})
	ry += 6

	// --- Collective AI ---
	printColoredAt(screen, locale.T("help.collective.title"), midX, ry, colorHelpAlgo)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.collective.1"),
		locale.T("help.collective.2"),
		locale.T("help.collective.3"),
		locale.T("help.collective.4"),
		locale.T("help.collective.5"),
		locale.T("help.collective.6"),
	})
	ry += 6

	// --- Fabrik-Modus (F5) ---
	printColoredAt(screen, locale.T("help.factory.title"), midX, ry, colorHelpFeature)
	ry += lineH
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.factory.1"),
		locale.T("help.factory.2"),
		locale.T("help.factory.3"),
		locale.T("help.factory.4"),
	})
	ry += 6

	// --- Tipps ---
	vector.StrokeLine(screen, float32(midX), float32(ry), float32(sw-px), float32(ry), 1, colorHelpSep, false)
	ry += 6
	printColoredAt(screen, locale.T("help.tips.title"), midX, ry, colorHelpNote)
	ry += lineH + 2
	helpParagraph(screen, midX, &ry, []string{
		locale.T("help.tips.1"),
		locale.T("help.tips.2"),
		locale.T("help.tips.3"),
		locale.T("help.tips.4"),
		locale.T("help.tips.5"),
		locale.T("help.tips.6"),
		locale.T("help.tips.7"),
		locale.T("help.tips.8"),
		locale.T("help.tips.9"),
		locale.T("help.tips.10"),
	})

	// Vertical separator between columns (up to where math section starts)
	ly := leftEndY
	mathStartY := ly
	if ry > ly {
		mathStartY = ry
	}
	mathStartY += 10
	vector.StrokeLine(screen, float32(midX-15), float32(y), float32(midX-15), float32(mathStartY-10), 1, colorHelpSep, false)

	// ========================
	// FULL-WIDTH: Mathematical Foundations
	// ========================
	my_ := mathStartY
	vector.StrokeLine(screen, float32(px), float32(my_), float32(sw-px), float32(my_), 1, colorHelpSep, false)
	my_ += 8

	// Section title
	mathTitle := locale.T("help.math.title")
	mathTitleW := runeLen(mathTitle) * charW
	printColoredAt(screen, mathTitle, sw/2-mathTitleW/2, my_, colorHelpSection)
	my_ += lineH + 2
	mathIntro := locale.T("help.math.intro")
	printColoredAt(screen, mathIntro, px+5, my_, colorHelpDim)
	my_ += lineH + 6

	// Two-column math layout
	mlx := px        // left math column
	mrx := sw/2 + 10 // right math column
	colW := sw/2 - 40
	_ = colW

	// Helper: draw one algorithm math block
	type mathBlock struct {
		name    string
		formula []string
		applied []string
	}

	mathAlgos := []mathBlock{
		{
			locale.T("help.math.gwo.name"),
			[]string{
				locale.T("help.math.gwo.formula.1"),
				locale.T("help.math.gwo.formula.2"),
				locale.T("help.math.gwo.formula.3"),
			},
			[]string{
				locale.T("help.math.gwo.applied.1"),
				locale.T("help.math.gwo.applied.2"),
				locale.T("help.math.gwo.applied.3"),
				locale.T("help.math.gwo.applied.4"),
			},
		},
		{
			locale.T("help.math.woa.name"),
			[]string{
				locale.T("help.math.woa.formula.1"),
				locale.T("help.math.woa.formula.2"),
				locale.T("help.math.woa.formula.3"),
			},
			[]string{
				locale.T("help.math.woa.applied.1"),
				locale.T("help.math.woa.applied.2"),
				locale.T("help.math.woa.applied.3"),
				locale.T("help.math.woa.applied.4"),
			},
		},
		{
			locale.T("help.math.cuckoo.name"),
			[]string{
				locale.T("help.math.cuckoo.formula.1"),
				locale.T("help.math.cuckoo.formula.2"),
				locale.T("help.math.cuckoo.formula.3"),
			},
			[]string{
				locale.T("help.math.cuckoo.applied.1"),
				locale.T("help.math.cuckoo.applied.2"),
				locale.T("help.math.cuckoo.applied.3"),
				locale.T("help.math.cuckoo.applied.4"),
			},
		},
		{
			locale.T("help.math.de.name"),
			[]string{
				locale.T("help.math.de.formula.1"),
				locale.T("help.math.de.formula.2"),
				locale.T("help.math.de.formula.3"),
			},
			[]string{
				locale.T("help.math.de.applied.1"),
				locale.T("help.math.de.applied.2"),
				locale.T("help.math.de.applied.3"),
				locale.T("help.math.de.applied.4"),
			},
		},
		{
			locale.T("help.math.sa.name"),
			[]string{
				locale.T("help.math.sa.formula.1"),
				locale.T("help.math.sa.formula.2"),
				locale.T("help.math.sa.formula.3"),
			},
			[]string{
				locale.T("help.math.sa.applied.1"),
				locale.T("help.math.sa.applied.2"),
				locale.T("help.math.sa.applied.3"),
				locale.T("help.math.sa.applied.4"),
			},
		},
		{
			locale.T("help.math.bfo.name"),
			[]string{
				locale.T("help.math.bfo.formula.1"),
				locale.T("help.math.bfo.formula.2"),
				locale.T("help.math.bfo.formula.3"),
			},
			[]string{
				locale.T("help.math.bfo.applied.1"),
				locale.T("help.math.bfo.applied.2"),
				locale.T("help.math.bfo.applied.3"),
				locale.T("help.math.bfo.applied.4"),
			},
		},
		{
			locale.T("help.math.abc.name"),
			[]string{
				locale.T("help.math.abc.formula.1"),
				locale.T("help.math.abc.formula.2"),
				locale.T("help.math.abc.formula.3"),
			},
			[]string{
				locale.T("help.math.abc.applied.1"),
				locale.T("help.math.abc.applied.2"),
				locale.T("help.math.abc.applied.3"),
				locale.T("help.math.abc.applied.4"),
			},
		},
		{
			locale.T("help.math.hso.name"),
			[]string{
				locale.T("help.math.hso.formula.1"),
				locale.T("help.math.hso.formula.2"),
				locale.T("help.math.hso.formula.3"),
			},
			[]string{
				locale.T("help.math.hso.applied.1"),
				locale.T("help.math.hso.applied.2"),
				locale.T("help.math.hso.applied.3"),
				locale.T("help.math.hso.applied.4"),
			},
		},
		{
			locale.T("help.math.mfo.name"),
			[]string{
				locale.T("help.math.mfo.formula.1"),
				locale.T("help.math.mfo.formula.2"),
				locale.T("help.math.mfo.formula.3"),
			},
			[]string{
				locale.T("help.math.mfo.applied.1"),
				locale.T("help.math.mfo.applied.2"),
				locale.T("help.math.mfo.applied.3"),
				locale.T("help.math.mfo.applied.4"),
			},
		},
		{
			locale.T("help.math.sca.name"),
			[]string{
				locale.T("help.math.sca.formula.1"),
				locale.T("help.math.sca.formula.2"),
				locale.T("help.math.sca.formula.3"),
			},
			[]string{
				locale.T("help.math.sca.applied.1"),
				locale.T("help.math.sca.applied.2"),
				locale.T("help.math.sca.applied.3"),
				locale.T("help.math.sca.applied.4"),
			},
		},
		{
			locale.T("help.math.hho.name"),
			[]string{
				locale.T("help.math.hho.formula.1"),
				locale.T("help.math.hho.formula.2"),
				locale.T("help.math.hho.formula.3"),
			},
			[]string{
				locale.T("help.math.hho.applied.1"),
				locale.T("help.math.hho.applied.2"),
				locale.T("help.math.hho.applied.3"),
				locale.T("help.math.hho.applied.4"),
			},
		},
		{
			locale.T("help.math.gsa.name"),
			[]string{
				locale.T("help.math.gsa.formula.1"),
				locale.T("help.math.gsa.formula.2"),
				locale.T("help.math.gsa.formula.3"),
			},
			[]string{
				locale.T("help.math.gsa.applied.1"),
				locale.T("help.math.gsa.applied.2"),
				locale.T("help.math.gsa.applied.3"),
				locale.T("help.math.gsa.applied.4"),
			},
		},
		{
			locale.T("help.math.fpa.name"),
			[]string{
				locale.T("help.math.fpa.formula.1"),
				locale.T("help.math.fpa.formula.2"),
				locale.T("help.math.fpa.formula.3"),
			},
			[]string{
				locale.T("help.math.fpa.applied.1"),
				locale.T("help.math.fpa.applied.2"),
				locale.T("help.math.fpa.applied.3"),
				locale.T("help.math.fpa.applied.4"),
			},
		},
		{
			locale.T("help.math.tlbo.name"),
			[]string{
				locale.T("help.math.tlbo.formula.1"),
				locale.T("help.math.tlbo.formula.2"),
				locale.T("help.math.tlbo.formula.3"),
			},
			[]string{
				locale.T("help.math.tlbo.applied.1"),
				locale.T("help.math.tlbo.applied.2"),
				locale.T("help.math.tlbo.applied.3"),
				locale.T("help.math.tlbo.applied.4"),
			},
		},
		{
			locale.T("help.math.bat.name"),
			[]string{
				locale.T("help.math.bat.formula.1"),
				locale.T("help.math.bat.formula.2"),
				locale.T("help.math.bat.formula.3"),
			},
			[]string{
				locale.T("help.math.bat.applied.1"),
				locale.T("help.math.bat.applied.2"),
				locale.T("help.math.bat.applied.3"),
				locale.T("help.math.bat.applied.4"),
			},
		},
		{
			locale.T("help.math.ssa.name"),
			[]string{
				locale.T("help.math.ssa.formula.1"),
				locale.T("help.math.ssa.formula.2"),
				locale.T("help.math.ssa.formula.3"),
			},
			[]string{
				locale.T("help.math.ssa.applied.1"),
				locale.T("help.math.ssa.applied.2"),
				locale.T("help.math.ssa.applied.3"),
				locale.T("help.math.ssa.applied.4"),
			},
		},
		{
			locale.T("help.math.eo.name"),
			[]string{
				locale.T("help.math.eo.formula.1"),
				locale.T("help.math.eo.formula.2"),
				locale.T("help.math.eo.formula.3"),
			},
			[]string{
				locale.T("help.math.eo.applied.1"),
				locale.T("help.math.eo.applied.2"),
				locale.T("help.math.eo.applied.3"),
				locale.T("help.math.eo.applied.4"),
			},
		},
		{
			locale.T("help.math.ao.name"),
			[]string{
				locale.T("help.math.ao.formula.1"),
				locale.T("help.math.ao.formula.2"),
				locale.T("help.math.ao.formula.3"),
			},
			[]string{
				locale.T("help.math.ao.applied.1"),
				locale.T("help.math.ao.applied.2"),
				locale.T("help.math.ao.applied.3"),
				locale.T("help.math.ao.applied.4"),
				locale.T("help.math.ao.applied.5"),
			},
		},
		{
			locale.T("help.math.da.name"),
			[]string{
				locale.T("help.math.da.formula.1"),
				locale.T("help.math.da.formula.2"),
				locale.T("help.math.da.formula.3"),
			},
			[]string{
				locale.T("help.math.da.applied.1"),
				locale.T("help.math.da.applied.2"),
				locale.T("help.math.da.applied.3"),
				locale.T("help.math.da.applied.4"),
			},
		},
		{
			locale.T("help.math.jaya.name"),
			[]string{
				locale.T("help.math.jaya.formula.1"),
				locale.T("help.math.jaya.formula.2"),
				locale.T("help.math.jaya.formula.3"),
			},
			[]string{
				locale.T("help.math.jaya.applied.1"),
				locale.T("help.math.jaya.applied.2"),
				locale.T("help.math.jaya.applied.3"),
				locale.T("help.math.jaya.applied.4"),
			},
		},
	}

	// Draw math blocks in two columns
	mlY := my_
	mrY := my_
	for i, mb := range mathAlgos {
		var cx int
		var cy *int
		if i%2 == 0 {
			cx = mlx
			cy = &mlY
		} else {
			cx = mrx
			cy = &mrY
		}

		// Algo name header
		printColoredAt(screen, mb.name, cx+5, *cy, colorHelpAlgoHdr)
		*cy += lineH + 1

		// Formulas (gold)
		for _, f := range mb.formula {
			printColoredAt(screen, "  "+f, cx+5, *cy, colorHelpMath)
			*cy += lineH
		}
		*cy += 2

		// Application (green)
		for _, a := range mb.applied {
			printColoredAt(screen, "  "+a, cx+5, *cy, colorHelpApply)
			*cy += lineH
		}
		*cy += 10
	}

	// Vertical separator in math section
	finalY := mlY
	if mrY > mlY {
		finalY = mrY
	}
	vector.StrokeLine(screen, float32(midX-15), float32(mathStartY), float32(midX-15), float32(finalY), 1, colorHelpSep, false)

	// General math concepts section
	finalY += 4
	vector.StrokeLine(screen, float32(px), float32(finalY), float32(sw-px), float32(finalY), 1, colorHelpSep, false)
	finalY += 8
	conceptTitle := locale.T("help.concepts.title")
	conceptTitleW := runeLen(conceptTitle) * charW
	printColoredAt(screen, conceptTitle, sw/2-conceptTitleW/2, finalY, colorHelpSection)
	finalY += lineH + 4

	// Left: Exploration vs Exploitation
	cly := finalY
	printColoredAt(screen, locale.T("help.explore_exploit.title"), px+5, cly, colorHelpAlgoHdr)
	cly += lineH
	helpParagraph(screen, px, &cly, []string{
		locale.T("help.explore_exploit.1"),
		locale.T("help.explore_exploit.2"),
		locale.T("help.explore_exploit.3"),
		locale.T("help.explore_exploit.4"),
		locale.T("help.explore_exploit.5"),
		locale.T("help.explore_exploit.6"),
		locale.T("help.explore_exploit.7"),
	})
	cly += 6
	printColoredAt(screen, locale.T("help.levy.title"), px+5, cly, colorHelpAlgoHdr)
	cly += lineH
	helpParagraph(screen, px, &cly, []string{
		locale.T("help.levy.1"),
		locale.T("help.levy.2"),
		locale.T("help.levy.3"),
		locale.T("help.levy.4"),
	})

	// Right: Convergence + No Free Lunch
	cry := finalY
	printColoredAt(screen, locale.T("help.convergence.title"), mrx+5, cry, colorHelpAlgoHdr)
	cry += lineH
	helpParagraph(screen, mrx, &cry, []string{
		locale.T("help.convergence.1"),
		locale.T("help.convergence.2"),
		locale.T("help.convergence.3"),
		locale.T("help.convergence.4"),
		locale.T("help.convergence.5"),
		locale.T("help.convergence.6"),
	})
	cry += 6
	printColoredAt(screen, locale.T("help.nfl.title"), mrx+5, cry, colorHelpAlgoHdr)
	cry += lineH
	helpParagraph(screen, mrx, &cry, []string{
		locale.T("help.nfl.1"),
		locale.T("help.nfl.2"),
		locale.T("help.nfl.3"),
		locale.T("help.nfl.4"),
		locale.T("help.nfl.5"),
		locale.T("help.nfl.6"),
	})

	endY := cly
	if cry > endY {
		endY = cry
	}
	// Separator between concept columns
	vector.StrokeLine(screen, float32(midX-15), float32(finalY), float32(midX-15), float32(endY), 1, colorHelpSep, false)

	endY += 10
	_ = endY

	// Credits
	printColoredAt(screen, "SwarmSim v2.1", midX, endY, color.RGBA{100, 120, 160, 200})
	endY += lineH
	printColoredAt(screen, locale.T("help.credits"), midX, endY, color.RGBA{80, 90, 110, 180})

	// Footer
	footerY := sh - 20
	footer := locale.T("help.footer")
	footerW := runeLen(footer) * charW
	vector.DrawFilledRect(screen, 0, float32(footerY-5), float32(sw), float32(lineH+10), color.RGBA{0, 0, 0, 240}, false)
	printColoredAt(screen, footer, sw/2-footerW/2, footerY, colorHelpDim)
}
