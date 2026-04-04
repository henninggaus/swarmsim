package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"swarmsim/locale"
)

// drawHelpLeftColumn draws the Quick Start + SwarmScript Reference (left column).
// Returns the final Y position after all left-column content.
func drawHelpLeftColumn(screen *ebiten.Image, px, midX, y int, scrollY int) int {
	ly := y

	// -- SCHNELLSTART --
	printColoredAt(screen, locale.T("help.quickstart.title"), px, ly, color.RGBA{120, 255, 120, 255})
	ly += lineH + 2
	helpParagraph(screen, px, &ly, []string{
		locale.T("help.quickstart.1"),
		locale.T("help.quickstart.2"),
		locale.T("help.quickstart.3"),
		locale.T("help.quickstart.4"),
		locale.T("help.quickstart.5"),
	})
	ly += 6

	// -- BEDIENUNG --
	printColoredAt(screen, locale.T("help.controls.title"), px, ly, colorHelpSection)
	ly += lineH + 2
	helpParagraph(screen, px, &ly, []string{
		locale.T("help.controls.1"),
		locale.T("help.controls.2"),
		locale.T("help.controls.3"),
		locale.T("help.controls.4"),
		locale.T("help.controls.5"),
		locale.T("help.controls.6"),
		locale.T("help.controls.7"),
		locale.T("help.controls.8"),
		locale.T("help.controls.9"),
		locale.T("help.controls.10"),
		locale.T("help.controls.11"),
		locale.T("help.controls.12"),
		locale.T("help.controls.13"),
		locale.T("help.controls.14"),
		locale.T("help.controls.15"),
		locale.T("help.controls.16"),
		locale.T("help.controls.17"),
	})
	ly += 2
	helpKV(screen, px, &ly, []kv{
		{locale.T("help.tabs.arena.key"), locale.T("help.tabs.arena.desc")},
		{locale.T("help.tabs.evo.key"), locale.T("help.tabs.evo.desc")},
		{locale.T("help.tabs.display.key"), locale.T("help.tabs.display.desc")},
		{locale.T("help.tabs.tools.key"), locale.T("help.tabs.tools.desc")},
		{locale.T("help.tabs.algo.key"), locale.T("help.tabs.algo.desc")},
	})
	ly += 8

	// -- SWARMSCRIPT --
	vector.StrokeLine(screen, float32(px), float32(ly), float32(midX-20), float32(ly), 1, colorHelpSep, false)
	ly += 6
	printColoredAt(screen, locale.T("help.swarmscript.title"), px, ly, colorHelpSection)
	ly += lineH + 2

	helpParagraph(screen, px, &ly, []string{
		locale.T("help.swarmscript.1"),
		locale.T("help.swarmscript.2"),
	})
	ly += 2

	printColoredAt(screen, locale.T("help.syntax.label"), px+5, ly, colorHelpDim)
	printColoredAt(screen, "IF", px+55, ly, colorHelpSyntax)
	printColoredAt(screen, locale.T("help.syntax.sensor_tpl"), px+55+3*charW, ly, colorHelpSensor)
	printColoredAt(screen, "THEN", px+55+24*charW, ly, colorHelpSyntax)
	printColoredAt(screen, locale.T("help.syntax.action_tpl"), px+55+29*charW, ly, colorHelpAction)
	ly += lineH

	printColoredAt(screen, locale.T("help.syntax.conditions"), px+5, ly, colorHelpDim)
	printColoredAt(screen, "IF ... AND ... AND ... THEN ...", px+130, ly, colorHelpSyntax)
	ly += lineH

	printColoredAt(screen, locale.T("help.syntax.evolvable"), px+5, ly, colorHelpDim)
	printColoredAt(screen, "$A:15", px+130, ly, colorHelpSensor)
	printColoredAt(screen, locale.T("help.syntax.evolvable_desc"), px+165, ly, colorHelpDim)
	ly += lineH

	printColoredAt(screen, locale.T("help.syntax.comments"), px+5, ly, colorHelpDim)
	printColoredAt(screen, locale.T("help.syntax.comments_desc"), px+130, ly, color.RGBA{100, 100, 110, 255})
	ly += lineH + 4

	// Example
	printColoredAt(screen, locale.T("help.example.title"), px+5, ly, colorHelpNote)
	ly += lineH
	exampleLines := []struct {
		text string
		clr  color.RGBA
	}{
		{"IF carry == 0 AND p_dist < 20 THEN PICKUP", colorHelpText},
		{locale.T("help.example.comment.1"), color.RGBA{90, 90, 100, 255}},
		{"IF match == 1 THEN GOTO_DROPOFF", colorHelpText},
		{locale.T("help.example.comment.2"), color.RGBA{90, 90, 100, 255}},
		{"IF near_dist < 15 THEN TURN_FROM_NEAREST", colorHelpText},
		{locale.T("help.example.comment.3"), color.RGBA{90, 90, 100, 255}},
		{"IF true THEN FWD", colorHelpText},
		{locale.T("help.example.comment.4"), color.RGBA{90, 90, 100, 255}},
	}
	for _, ex := range exampleLines {
		printColoredAt(screen, "  "+ex.text, px+5, ly, ex.clr)
		ly += lineH
	}
	ly += 6

	// -- SENSOREN (complete) --
	printColoredAt(screen, locale.T("help.sensors.title"), px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVSensor(screen, px, &ly, []kv{
		{"near_dist", locale.T("help.sensor.near_dist")},
		{"neighbors", locale.T("help.sensor.neighbors")},
		{"carry", locale.T("help.sensor.carry")},
		{"match", locale.T("help.sensor.match")},
		{"p_dist", locale.T("help.sensor.p_dist")},
		{"d_dist", locale.T("help.sensor.d_dist")},
		{"has_pkg", locale.T("help.sensor.has_pkg")},
		{"light", locale.T("help.sensor.light")},
		{"obs_ahead", locale.T("help.sensor.obs_ahead")},
		{"wall_right", locale.T("help.sensor.wall_right")},
		{"wall_left", locale.T("help.sensor.wall_left")},
		{"edge", locale.T("help.sensor.edge")},
		{"rnd", locale.T("help.sensor.rnd")},
		{"tick", locale.T("help.sensor.tick")},
		{"state", locale.T("help.sensor.state")},
		{"counter", locale.T("help.sensor.counter")},
		{"heading", locale.T("help.sensor.heading")},
		{"team", locale.T("help.sensor.team")},
		{"team_score", locale.T("help.sensor.team_score")},
		{"enemy_score", locale.T("help.sensor.enemy_score")},
		{"msg", locale.T("help.sensor.msg")},
		{"on_ramp", locale.T("help.sensor.on_ramp")},
		{"truck_here", locale.T("help.sensor.truck_here")},
		{"truck_pkg", locale.T("help.sensor.truck_pkg")},
		{"speed", locale.T("help.sensor.speed")},
		{"bot_ahead", locale.T("help.sensor.bot_ahead")},
		{"bot_behind", locale.T("help.sensor.bot_behind")},
		{"bot_left", locale.T("help.sensor.bot_left")},
		{"bot_right", locale.T("help.sensor.bot_right")},
		{"visited_here", locale.T("help.sensor.visited_here")},
		{"visited_ahead", locale.T("help.sensor.visited_ahead")},
		{"explored", locale.T("help.sensor.explored")},
		{"group_carry", locale.T("help.sensor.group_carry")},
		{"group_speed", locale.T("help.sensor.group_speed")},
		{"group_size", locale.T("help.sensor.group_size")},
	})
	ly += 6

	// -- AKTIONEN (complete) --
	printColoredAt(screen, locale.T("help.actions.title"), px, ly, colorHelpSection)
	ly += lineH + 2
	helpKVAction(screen, px, &ly, []kv{
		{"FWD", locale.T("help.action.fwd")},
		{"STOP", locale.T("help.action.stop")},
		{"TURN_RIGHT N", locale.T("help.action.turn_right")},
		{"TURN_LEFT N", locale.T("help.action.turn_left")},
		{"TURN_RANDOM", locale.T("help.action.turn_random")},
		{"TURN_TO_NEAREST", locale.T("help.action.turn_to_nearest")},
		{"TURN_FROM_NEAREST", locale.T("help.action.turn_from_nearest")},
		{"TURN_TO_LIGHT", locale.T("help.action.turn_to_light")},
		{"TURN_TO_CENTER", locale.T("help.action.turn_to_center")},
		{"FOLLOW_NEAREST", locale.T("help.action.follow_nearest")},
		{"PICKUP", locale.T("help.action.pickup")},
		{"DROP", locale.T("help.action.drop")},
		{"GOTO_DROPOFF", locale.T("help.action.goto_dropoff")},
		{"GOTO_PICKUP", locale.T("help.action.goto_pickup")},
		{"AVOID_OBSTACLE", locale.T("help.action.avoid_obstacle")},
		{"WALL_FOLLOW_RIGHT", locale.T("help.action.wall_follow_right")},
		{"WALL_FOLLOW_LEFT", locale.T("help.action.wall_follow_left")},
		{"SET_LED R G B", locale.T("help.action.set_led")},
		{"COPY_LED", locale.T("help.action.copy_led")},
		{"SEND_MESSAGE N", locale.T("help.action.send_message")},
		{"SET_STATE N", locale.T("help.action.set_state")},
		{"INC_COUNTER", locale.T("help.action.inc_counter")},
		{"RESET_COUNTER", locale.T("help.action.reset_counter")},
		{"GOTO_BEACON", locale.T("help.action.goto_beacon")},
	})

	return ly
}
