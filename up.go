package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Config struct {
	Lists []List `json:"lists"`
	About struct{}
}

type List struct {
	Name string `json:"name"`
	Item []Item `json:"item"`
}

type Item struct {
	ItemName string    `json:"item_name"`
	ItemDes  string    `json:"item_des"`
	ItemRun  string    `json:"item_run"`
	SubItems []SubItem `json:"sub_items,omitempty"`
}

type SubItem struct {
	ItemName string `json:"item_name"`
	ItemDes  string `json:"item_des"`
	ItemRun  string `json:"item_run"`
}

var config Config

func main() {
	a := app.New()
	w := a.NewWindow("Config Editor")

	loadConfig()

	listView := createListView(w)
	detailView := container.NewVBox()

	split := container.NewHSplit(listView, detailView)
	split.Offset = 0.3

	w.SetContent(split)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}

func createListView(w fyne.Window) *fyne.Container {
	list := widget.NewList(
		func() int { return len(config.Lists) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(config.Lists[id].Name)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		showListDetails(w, id)
	}

	addButton := widget.NewButton("Add List", func() {
		addNewList(w)
	})

	return container.NewBorder(nil, addButton, nil, nil, list)
}

func showListDetails(w fyne.Window, id widget.ListItemID) {
	list := &config.Lists[id]

	nameEntry := widget.NewEntry()
	nameEntry.SetText(list.Name)

	itemList := widget.NewList(
		func() int { return len(list.Item) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(list.Item[id].ItemName)
		},
	)

	itemList.OnSelected = func(itemID widget.ListItemID) {
		showItemDetails(w, id, itemID)
	}

	addItemButton := widget.NewButton("Add Item", func() {
		addNewItem(w, id)
	})

	saveButton := widget.NewButton("Save", func() {
		list.Name = nameEntry.Text
		saveConfig()
		refreshUI(w)
	})

	deleteButton := widget.NewButton("Delete List", func() {
		dialog.ShowConfirm("Delete List", "Are you sure you want to delete this list?", func(b bool) {
			if b {
				config.Lists = append(config.Lists[:id], config.Lists[id+1:]...)
				saveConfig()
				showListDetails(w, 0) // Show first list after deletion
				refreshUI(w)
			}
		}, w)
	})

	content := container.NewBorder(
		container.NewVBox(nameEntry, saveButton, deleteButton),
		addItemButton,
		nil,
		nil,
		itemList,
	)

	updateDetailView(w, content)
	refreshUI(w)
}

func showItemDetails(w fyne.Window, listID, itemID widget.ListItemID) {
	item := &config.Lists[listID].Item[itemID]

	nameEntry := widget.NewEntry()
	nameEntry.SetText(item.ItemName)

	desEntry := widget.NewMultiLineEntry()
	desEntry.SetText(item.ItemDes)

	runEntry := widget.NewEntry()
	runEntry.SetText(item.ItemRun)

	saveButton := widget.NewButton("Save", func() {
		item.ItemName = nameEntry.Text
		item.ItemDes = desEntry.Text
		item.ItemRun = runEntry.Text
		saveConfig()
		refreshUI(w)
	})

	deleteButton := widget.NewButton("Delete Item", func() {
		dialog.ShowConfirm("Delete Item", "Are you sure you want to delete this item?", func(b bool) {
			if b {
				config.Lists[listID].Item = append(config.Lists[listID].Item[:itemID], config.Lists[listID].Item[itemID+1:]...)
				saveConfig()
				showListDetails(w, listID)
				refreshUI(w)
			}
		}, w)
	})

	// Create a VBox to hold all sub-items
	subItemsContainer := container.NewVBox()

	// Add all sub-items to the container
	for i, subItem := range item.SubItems {
		subItemCard := widget.NewCard(
			subItem.ItemName,
			"",
			container.NewVBox(
				widget.NewLabel("Description: "+subItem.ItemDes),
				widget.NewLabel("Run Command: "+subItem.ItemRun),
				widget.NewButton("Delete Sub Item", func(index int) func() {
					return func() {
						item.SubItems = append(item.SubItems[:index], item.SubItems[index+1:]...)
						saveConfig()
						showItemDetails(w, listID, itemID)
						refreshUI(w)
					}
				}(i)),
			),
		)
		subItemsContainer.Add(subItemCard)
	}

	// Wrap the sub-items container in a scroll container
	subItemsScroll := container.NewScroll(subItemsContainer)
	subItemsScroll.SetMinSize(fyne.NewSize(300, 200))

	addSubItemButton := widget.NewButton("Add Sub Item", func() {
		addNewSubItem(w, listID, itemID)
	})

	content := container.NewVBox(
		nameEntry,
		desEntry,
		runEntry,
		saveButton,
		deleteButton,
		widget.NewLabel("Sub Items:"),
		subItemsScroll,
		addSubItemButton,
	)

	updateDetailView(w, content)
	refreshUI(w)
}

func updateDetailView(w fyne.Window, content fyne.CanvasObject) {
	split := w.Content().(*container.Split)
	split.Trailing = content
	w.Content().Refresh()
}

func addNewList(w fyne.Window) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter list name")

	dialog.ShowForm("Add New List", "Add", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}, func(b bool) {
		if b {
			newList := List{Name: nameEntry.Text, Item: []Item{}}
			config.Lists = append(config.Lists, newList)
			saveConfig()
			refreshUI(w)
		}
	}, w)
}

func addNewItem(w fyne.Window, listID widget.ListItemID) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter item name")

	desEntry := widget.NewMultiLineEntry()
	desEntry.SetPlaceHolder("Enter item description")
	desEntry.SetMinRowsVisible(3)

	runEntry := widget.NewEntry()
	runEntry.SetPlaceHolder("Enter item run command")

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Description", desEntry),
			widget.NewFormItem("Run Command", runEntry),
		},
		OnSubmit: func() {
			newItem := Item{
				ItemName: nameEntry.Text,
				ItemDes:  desEntry.Text,
				ItemRun:  runEntry.Text,
			}
			config.Lists[listID].Item = append(config.Lists[listID].Item, newItem)
			saveConfig()
			showListDetails(w, listID)
			refreshUI(w)
		},
	}

	content := container.NewVBox(
		widget.NewLabel("Add New Item"),
		form,
	)

	dialog := dialog.NewCustom("Add New Item", "Cancel", content, w)
	dialog.SetOnClosed(func() {
		refreshUI(w)
	})

	dialog.Resize(fyne.NewSize(400, 300))

	dialog.Show()
}

func addNewSubItem(w fyne.Window, listID, itemID widget.ListItemID) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter sub-item name")

	desEntry := widget.NewMultiLineEntry()
	desEntry.SetPlaceHolder("Enter sub-item description")
	desEntry.SetMinRowsVisible(3)

	runEntry := widget.NewEntry()
	runEntry.SetPlaceHolder("Enter sub-item run command")

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Description", desEntry),
			widget.NewFormItem("Run Command", runEntry),
		},
		OnSubmit: func() {
			newSubItem := SubItem{
				ItemName: nameEntry.Text,
				ItemDes:  desEntry.Text,
				ItemRun:  runEntry.Text,
			}
			config.Lists[listID].Item[itemID].SubItems = append(config.Lists[listID].Item[itemID].SubItems, newSubItem)
			saveConfig()
			showItemDetails(w, listID, itemID)
			refreshUI(w)
		},
	}

	content := container.NewVBox(
		widget.NewLabel("Add New Sub Item"),
		form,
	)

	dialog := dialog.NewCustom("Add New Sub Item", "Cancel", content, w)
	dialog.SetOnClosed(func() {
		refreshUI(w)
	})

	dialog.Resize(fyne.NewSize(400, 300))

	dialog.Show()
}

func loadConfig() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
}

func saveConfig() {
	file, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	err = ioutil.WriteFile("config.json", file, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}
}

func refreshUI(w fyne.Window) {
	w.Content().Refresh()
	split := w.Content().(*container.Split)
	split.Leading.Refresh()
	split.Trailing.Refresh()
}
