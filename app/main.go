package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	ui "github.com/matthiasBender/hypertonia/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	badger "github.com/dgraph-io/badger/v4"
)

type BpRecord struct {
	Ts    time.Time
	Sys   byte
	Dia   byte
	Pulse byte
}

var (
	Records         []*BpRecord
	MutRecords      = sync.Mutex{}
	dbPrefixRecords = []byte("bpnRecord:")
)

func main() {
	a := app.NewWithID("mbender.hypertonia")
	w := a.NewWindow("Hypertonia")

	db := connectToDb(a.Storage().RootURI().Path() + "/badger")
	defer db.Close()

	db.View(readStoredRecords)
	log.Println(Records)

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

	form := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{Text: "Datum", Widget: dateInput},
			{Text: "Zeit", Widget: ui.CreateTimeEntry()},
			{Text: "Systolisch", Widget: sysInput},
			{Text: "Diastolisch", Widget: diaInput},
			{Text: "Puls", Widget: bpInput},
		},
		Orientation: widget.Horizontal,
	}
	form.OnSubmit = func() {
		record := &BpRecord{
			Ts:    time.Now(),
			Sys:   fetchFromEntry(sysInput),
			Dia:   fetchFromEntry(diaInput),
			Pulse: fetchFromEntry(bpInput),
		}
		MutRecords.Lock()
		defer MutRecords.Unlock()
		if err := db.Update(func(txn *badger.Txn) error {
			timestamp := strconv.FormatInt(record.Ts.Unix(), 10)
			jRecord, err := json.Marshal(record)
			if err != nil {
				return fmt.Errorf("cannot serialize json: %v", err)
			}

			return txn.Set(append(dbPrefixRecords, []byte(timestamp)...), jRecord)
		}); err != nil {
			log.Println("failed write data:", err)
			return
		}
		Records = append(Records, record)

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

func connectToDb(path string) *badger.DB {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		panic(err)
	}
	return db
}

func readStoredRecords(txn *badger.Txn) error {
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	MutRecords.Lock()
	defer MutRecords.Unlock()
	for it.Seek(dbPrefixRecords); it.ValidForPrefix(dbPrefixRecords); it.Next() {
		item := it.Item()
		record := &BpRecord{}
		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, record)
		}); err != nil {
			return fmt.Errorf("cannot read value: %v", err)
		}
		Records = append(Records, record)
	}

	return nil
}

func (r BpRecord) String() string {
	return fmt.Sprintf("%s - Sys: %d, Dia: %d, Pulse: %d", r.Ts.UTC().Format("2006-01-02 15:04:05"), r.Sys, r.Dia, r.Pulse)
}
