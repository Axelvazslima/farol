package main

import (
	"context"
	"docker-monitoring-ui/internal/entities"
	"docker-monitoring-ui/internal/services"
	"fmt"
	"github.com/rivo/tview"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var grid *tview.Grid

func main() {
	ctx := context.Background()
	service := &services.ContainerService{}
	app := tview.NewApplication()

	var updateViews func()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		app.Stop()
		os.Exit(0)
	}()

	runningView := tview.NewTextView()
	runningView.SetBorder(true).
		SetTitle("Running Containers")
	allView := tview.NewTextView()
	allView.SetBorder(true).
		SetTitle("All Containers")
	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[blue]Ready")

	updateViews = func() {
		service.Init(ctx)
		var runningBuffer, allBuffer strings.Builder
		for _, c := range service.GetAllContainers() {
			if c.Running {
				runningBuffer.WriteString(c.Name + "\n")
			}
			allBuffer.WriteString(fmt.Sprintf("%s (%s)\n",
				c.Name,
				map[bool]string{true: "Running", false: "Stopped"}[c.Running],
			))
		}
		runningView.SetText(runningBuffer.String())
		allView.SetText(allBuffer.String())
		statusBar.SetText("[blue]Refreshed")
	}

	startBtn := tview.NewButton("Start").SetSelectedFunc(func() {
		showContainerActionForm(app, "Start Container", getStoppedContainers(service), func(id string) error {
			return service.StartContainer(ctx, id)
		}, updateViews)
	})

	inspectBtn := tview.NewButton("Inspect").SetSelectedFunc(func() {
		showInspectForm(app, service)
	})

	stopBtn := tview.NewButton("Stop").SetSelectedFunc(func() {
		showContainerActionForm(app, "Stop Container", getRunningContainers(service), func(id string) error {
			return service.StopContainer(ctx, id)
		}, updateViews)
	})

	createBtn := tview.NewButton("Create").SetSelectedFunc(func() {
		showCreateForm(app, service, ctx, updateViews)
	})

	removeBtn := tview.NewButton("Remove").SetSelectedFunc(func() {
		showContainerActionForm(app, "Remove Container", service.GetAllContainers(), func(id string) error {
			return service.RemoveContainer(ctx, id)
		}, updateViews)
	})

	buttonGrid := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(startBtn, 0, 1, false).
		AddItem(stopBtn, 0, 1, false).
		AddItem(createBtn, 0, 1, false).
		AddItem(inspectBtn, 0, 1, false).
		AddItem(removeBtn, 0, 1, false)

	grid = tview.NewGrid().
		SetRows(3, 0, 3, 1).
		SetColumns(0, 0).
		AddItem(tview.NewTextView().SetText("Farol - 🐳 Docker Manager").SetTextAlign(tview.AlignCenter), 0, 0, 1, 2, 0, 0, false).
		AddItem(runningView, 1, 0, 1, 1, 0, 0, false).
		AddItem(allView, 1, 1, 1, 1, 0, 0, false).
		AddItem(buttonGrid, 2, 0, 1, 2, 0, 0, true).
		AddItem(statusBar, 3, 0, 1, 2, 0, 0, false)

	updateViews()

	if err := app.SetRoot(grid, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}

}

func showCreateForm(app *tview.Application, service *services.ContainerService, ctx context.Context, update func()) {
	form := tview.NewForm()

	var containerName string
	var imageName string

	form.AddInputField("Container Name", "", 20, nil, func(text string) {
		containerName = text
	})
	form.AddInputField("Image", "", 20, nil, func(text string) {
		imageName = text
	})

	form.AddButton("Create", func() {
		if containerName == "" || imageName == "" {
			showErrorModal(app, "Validation Error", "Both fields are required")
			return
		}
		if err := service.CreateContainer(ctx, containerName, imageName); err != nil {
			showErrorModal(app, "Create Error", err.Error())
		} else {
			update()
			app.SetRoot(grid, true)
		}
	})

	form.AddButton("Cancel", func() {
		app.SetRoot(grid, true)
	})

	form.SetBorder(true).SetTitle("Create Container").SetTitleAlign(tview.AlignLeft)
	app.SetRoot(form, true)
}

func showContainerActionForm(app *tview.Application, title string, containers []entities.Container, action func(string) error, update func()) {
	form := tview.NewForm()
	ids := make([]string, len(containers))
	names := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
		names[i] = c.Name
	}
	var selectedIdx int
	form.AddDropDown("Container", names, 0, func(option string, index int) {
		selectedIdx = index
	})
	form.AddButton("Confirm", func() {
		if selectedIdx >= 0 && selectedIdx < len(ids) {
			if err := action(ids[selectedIdx]); err != nil {
				showErrorModal(app, title+" Error", err.Error())
			} else {
				update()
				app.SetRoot(grid, true)
			}
		}
	})

	form.AddButton("Cancel", func() {
		app.SetRoot(grid, true)
	})
	form.SetBorder(true).SetTitle(title)
	app.SetRoot(form, true)
}

func showErrorModal(app *tview.Application, title, msg string) {
	modal := tview.NewModal().
		SetText("[red]" + msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(i int, l string) {
			app.SetRoot(grid, true)
		})
	modal.SetTitle(title)
	app.SetRoot(modal, false)
}

func getRunningContainers(service *services.ContainerService) []entities.Container {
	running := []entities.Container{}
	for _, c := range service.GetAllContainers() {
		if c.Running {
			running = append(running, c)
		}
	}
	return running
}

func getStoppedContainers(service *services.ContainerService) []entities.Container {
	stopped := []entities.Container{}
	for _, c := range service.GetAllContainers() {
		if !c.Running {
			stopped = append(stopped, c)
		}
	}
	return stopped
}

func displayContainerDetails(app *tview.Application, c entities.Container) {
	details := fmt.Sprintf("Name: %s\nID: %s\nImage: %s\nStatus: %s", c.Name, c.ID, c.Image, map[bool]string{true: "Running", false: "Stopped"}[c.Running])
	detailModal := tview.NewModal().SetText(details).AddButtons([]string{"OK"}).SetDoneFunc(func(_ int, _ string) {
		app.SetRoot(grid, true)
	})
	detailModal.SetTitle("Container Details")
	app.SetRoot(detailModal, false)
}

func showInspectForm(app *tview.Application, svc *services.ContainerService) {
	containers := svc.GetAllContainers()
	ids := make([]string, len(containers))
	names := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
		names[i] = c.Name
	}
	var idx int
	form := tview.NewForm()
	form.AddDropDown("Container", names, 0, func(_ string, i int) { idx = i })
	form.AddButton("Inspect", func() {
		displayContainerDetails(app, containers[idx])
	})
	form.AddButton("Cancel", func() { app.SetRoot(grid, true) })
	form.SetBorder(true).SetTitle("Inspect Container")
	app.SetRoot(form, true)
}
