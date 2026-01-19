package main

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/matthiasBender/hypertonia/storage"
	"github.com/matthiasBender/hypertonia/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.NewWithID("mbender.hypertonia")
	w := a.NewWindow("Hypertonia")

	err := storage.Connect(a.Storage().RootURI().Path() + "/badger")
	if err != nil {
		panic(err)
	}
	defer storage.Close()

	log.Println(storage.Read())

	sysInput := widget.NewEntry()
	sysInput.SetPlaceHolder("systolischer...")
	sysInput.MultiLine = false
	sysInput.Validator = createNumberValidator(90, 200, false)

	diaInput := widget.NewEntry()
	diaInput.SetPlaceHolder("diastolischer...")
	diaInput.MultiLine = false
	diaInput.Validator = createNumberValidator(50, 160, false)

	bpInput := widget.NewEntry()
	bpInput.SetPlaceHolder("in B/Min")
	bpInput.MultiLine = false
	bpInput.Validator = createNumberValidator(40, 120, true)

	now := time.Now()
	dateInput := widget.NewDateEntry()
	dateInput.SetDate(&now)

	timeInput := ui.CreateTimeEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "Datum", Widget: dateInput},
			{Text: "Zeit", Widget: timeInput.Entry},
			{Text: "Systolisch", Widget: sysInput},
			{Text: "Diastolisch", Widget: diaInput},
			{Text: "Puls", Widget: bpInput},
		},
		Orientation: widget.Horizontal,
	}
	form.OnSubmit = func() {
		t := time.Date(
			dateInput.Date.Year(),
			dateInput.Date.Month(),
			dateInput.Date.Day(),
			timeInput.Timestamp.Hour(),
			timeInput.Timestamp.Minute(), 0, 0,
			dateInput.Date.Location(),
		)
		record := &storage.BpRecord{
			Ts:    t,
			Sys:   fetchFromEntry(sysInput),
			Dia:   fetchFromEntry(diaInput),
			Pulse: fetchFromEntry(bpInput),
		}
		if err := storage.Save(record); err != nil {
			log.Println("failed write data:", err)
			return
		}

		// TODO: Write data into database!
		log.Println("inserted...", record)
		sysInput.Text = ""
		diaInput.Text = ""
		bpInput.Text = ""
		form.Refresh()
	}
	form.Refresh()

	grid := container.New(layout.NewVBoxLayout(), form)
	w.SetContent(grid)
	w.ShowAndRun()
}

func createNumberValidator(start, end int, optional bool) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			if optional {
				return nil
			}
			return errors.New("is empty")
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return errors.New("not a number")
		}
		if i < start || i > end {
			return errors.New("out of range")
		}
		return nil
	}
}

func fetchFromEntry(e *widget.Entry) byte {
	s := strings.TrimSpace(e.Text)
	if s == "" {
		return 0
	}
	result, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		log.Panicf("failed to convert %q to number: %v", s, err)
	}
	return byte(result)
}
