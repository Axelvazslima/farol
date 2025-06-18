package main

import (
	"context"
	"docker-monitoring-ui/internal/services"
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func start() {
	ctx := context.Background()
	service := &services.ContainerService{}
	service.Init(ctx)

	app := tview.NewApplication()

	runningView := tview.NewTextView()
	runningView.SetScrollable(true).
		SetBorder(true).
		SetTitle("Running Containers")

	allView := tview.NewTextView()
	allView.SetScrollable(true).
		SetBorder(true).
		SetTitle("All Containers")

	startView := tview.NewList()
	startView.SetBorder(true).
		SetTitle("Started Containers")

	stopView := tview.NewList()
	stopView.SetBorder(true).
		SetTitle("Stopped Containers")

	var updateViews func()
	updateViews = func() {
		service.Init(ctx)

		startView.Clear()
		stopView.Clear()

		var runningBuffer, allBuffer strings.Builder

		for _, c := range service.GetAllContainers() {
			status := "Stopped"
			if c.Running {
				status = "Running"
			}

			// HANDLED ERROR: Fprintf to a strings.Builder never fails,
			// but we use '_' to explicitly acknowledge and ignore the nil error.
			_, _ = fmt.Fprintf(&allBuffer, "%s (%s)\n", c.Name, status)
			if c.Running {
				_, _ = fmt.Fprintf(&runningBuffer, "%s\n", c.Name)
			}

			container := c
			if container.Running {
				stopView.AddItem(container.Name, "", 0, func() {
					// HANDLED ERROR: Check for errors when performing actions.
					if err := service.StopContainer(ctx, container.ID); err != nil {
						// For now, we log the error. In a real app, you might show a dialog.
						log.Printf("Failed to stop container %s: %v", container.Name, err)
					}
					app.QueueUpdateDraw(updateViews)
				})
			} else {
				startView.AddItem(container.Name, "", 0, func() {
					// HANDLED ERROR: Check for errors when performing actions.
					if err := service.StartContainer(ctx, container.ID); err != nil {
						log.Printf("Failed to start container %s: %v", container.Name, err)
					}
					app.QueueUpdateDraw(updateViews)
				})
			}
		}

		runningView.SetText(runningBuffer.String())
		allView.SetText(allBuffer.String())
	}

	updateViews()

	grid := tview.NewGrid().
		SetRows(0, 0).
		SetColumns(0, 0).
		SetBorders(true).
		AddItem(runningView, 0, 0, 1, 1, 0, 0, false).
		AddItem(allView, 0, 1, 1, 1, 0, 0, false).
		AddItem(startView, 1, 0, 1, 1, 0, 0, true).
		AddItem(stopView, 1, 1, 1, 1, 0, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if startView.HasFocus() {
				app.SetFocus(stopView)
			} else {
				app.SetFocus(startView)
			}
			return nil
		}
		return event
	})

	if err := app.SetRoot(grid, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func main() {
	// You might want to configure the log output, for example:
	// log.SetOutput(os.Stderr)
	start()
}
