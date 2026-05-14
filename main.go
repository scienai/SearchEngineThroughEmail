package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/textcore"
)

func main() {
	aiwin := core.NewBody()
	mainfrm := core.NewFrameRow(aiwin)

	memubarfrm := core.NewFrameRow(mainfrm)
	menu := func(m *core.Scene) {
		m1 := core.NewButton(m).SetText("Exit")
		m1.SetTooltip("Exit this program")
		m1.OnClick(func(e events.Event) {
			aiwin.Close()
		})
	}
	core.NewButton(memubarfrm).SetIcon(icons.Menu).SetMenu(menu)

	searchbarfrm := core.NewFrameRow(mainfrm)
	emailPassword := core.NewTextField(searchbarfrm)
	emailPassword.SetTypePassword()
	emailPassword.SetPlaceholder("Email Password")
	searchText := core.NewTextField(searchbarfrm)
	searchText.SetTypePassword()
	searchText.SetPlaceholder("Search text")
	core.NewButton(searchbarfrm).SetIcon(icons.Search).OnClick(func(e events.Event) {

	})

	resultfrm := core.NewFrameRow(mainfrm)
	inputeditor := textcore.NewEditor(resultfrm)
	inputeditor.Lines.SetTextLines(nil)

	bottomfrm := core.NewFrameRow(mainfrm)
	message := core.NewText(bottomfrm)
	message.SetText("")

	aiwin.RunMainWindow()
}
