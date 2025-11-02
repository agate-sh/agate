package main

import (
    "fmt"
    "agate/pkg/gui/components"
    "github.com/charmbracelet/lipgloss"
)

func main() {
    availableHeight := 40
    agentsContent := components.ApplyPaneContentPadding("agents", 50)
    frameHeight := components.PaneBaseStyle.GetVerticalFrameSize()
    leftContentHeight := availableHeight - frameHeight
    if leftContentHeight < 1 {
        leftContentHeight = 1
    }
    agentsWrapped := lipgloss.NewStyle().Width(52).MaxHeight(leftContentHeight).Render(agentsContent)
    agentsContentAligned := lipgloss.PlaceVertical(leftContentHeight, lipgloss.Top, agentsWrapped)

    pane := components.PaneBaseStyle.Height(availableHeight - 2).Render(agentsContentAligned)
    fmt.Println("height", lipgloss.Height(pane))
}
