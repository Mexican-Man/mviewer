package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"

	"github.com/Mexican-Man/tui-go"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	var file string
	var uri string

	cmd := &cobra.Command{
		Short: "mview is a simple terminal browser for MongoDB",
		Long:  "TODO",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if file or uri is given
			if file == "" && uri == "" {
				return fmt.Errorf("no file or uri given")
			} else if file != "" && uri != "" {
				return fmt.Errorf("only one of file or uri can be given")
			}

			// If file, get uri from file
			if file != "" {
				b, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				uri = string(b)
			}

			// Attempt to connect to db
			client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
			if err != nil {
				return fmt.Errorf("failed to connect to mongodb: %v", err)
			}

			// Store the currently viewed items
			var currTab int

			// Read databases from db
			dbs, err := client.ListDatabaseNames(context.TODO(), bson.D{}, options.ListDatabases().SetNameOnly(true))
			if err != nil {
				return fmt.Errorf("failed to list databases: %v", err)
			}

			// Declare the main bars
			var databaseBar, collectionBar, documentsBar *tui.List

			// Declare document filter
			var documentFilter bson.M

			// Add documentsBar
			documentsBar = tui.NewList()
			documentsScroll := tui.NewScrollArea(documentsBar, image.Point{15, 0})
			documentBox := tui.NewVBox(documentsScroll)
			documentBox.SetBorder(true)

			// Add collections
			collectionBar = tui.NewList()
			collectionBarScroll := tui.NewScrollArea(collectionBar, image.Point{15, 0})
			collectionBarBox := tui.NewVBox(collectionBarScroll)
			collectionBarBox.SetBorder(true)
			collectionBar.OnSelectionChanged(func(*tui.List) {
				// Catch for when we populate the list, but nothing is selected yet
				if collectionBar.Selected() < 0 {
					return
				}
				cursor, err := client.Database(databaseBar.SelectedItem()).Collection(collectionBar.SelectedItem()).Find(context.TODO(), documentFilter, options.Find().SetLimit(400))
				if err != nil {
					panic(err)
				}
				documentsBar.RemoveItems()
				for cursor.Next(context.TODO()) {
					var doc bson.M
					err = cursor.Decode(&doc)
					if err != nil {
						panic(err)
					}

					// Parse BSON to regular JSON
					documentsBar.AddItems(cursor.Current.String())
				}
			})

			// Populate databases
			databaseBar = tui.NewList()
			for _, db := range dbs {
				databaseBar.AddItems(db)
			}

			// Add databaseBar
			databaseBarScroll := tui.NewScrollArea(databaseBar, image.Point{15, 0})
			databaseBarBox := tui.NewVBox(databaseBarScroll)
			databaseBarBox.SetBorder(true)
			databaseBar.OnSelectionChanged(func(*tui.List) {
				cursor, err := client.Database(databaseBar.SelectedItem()).ListCollections(context.TODO(), bson.D{}, options.ListCollections())
				if err != nil {
					panic(err)
				}
				documentsBar.RemoveItems()
				collectionBar.RemoveItems()
				for cursor.Next(context.TODO()) {
					var doc bson.M
					err = cursor.Decode(&doc)
					if err != nil {
						panic(err)
					}
					collectionBar.AddItems(cursor.Current.Lookup("name").StringValue())
				}
			})

			input := tui.NewEntry()
			input.SetSizePolicy(tui.Expanding, tui.Maximum)

			inputBox := tui.NewHBox(input)
			inputBox.SetBorder(true)
			inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

			filterBar := tui.NewVBox(documentBox, inputBox)
			filterBar.SetSizePolicy(tui.Expanding, tui.Expanding)

			// Marshal text into BSON and use that to filter the documents
			input.OnSubmit(func(e *tui.Entry) {
				documentFilter = bson.M{}
				bson.UnmarshalExtJSON([]byte(e.Text()), true, &documentFilter)
				collectionBar.Select(collectionBar.Selected())
				documentsBar.Select(-1)
			})

			helpText := tui.NewLabel("Press ESC to exit, use arrow keys to navigate. Hit RIGHT more to enter the filter box, then press enter to search.")
			helpTextBar := tui.NewHBox(helpText)
			helpTextBar.SetSizePolicy(tui.Expanding, tui.Maximum)

			root := tui.NewVBox(tui.NewHBox(databaseBarBox, collectionBarBox, filterBar), helpTextBar)
			root.SetSizePolicy(tui.Expanding, tui.Expanding)

			ui, err := tui.New(root)
			if err != nil {
				log.Fatal(err)
			}

			// Set up key bindings for each tab
			tabsArray := []*tui.List{databaseBar, collectionBar, documentsBar}
			scrollsArray := []*tui.ScrollArea{databaseBarScroll, collectionBarScroll, documentsScroll}
			ui.SetKeybinding("Esc", func() { ui.Quit() })
			ui.SetKeybinding("Left", func() {
				if currTab > 0 {
					tabsArray[currTab].Select(-1)
					currTab--
				}
			})
			ui.SetKeybinding("Right", func() {
				if currTab == len(scrollsArray)-1 {
					input.SetFocused(true)
				} else if currTab < len(scrollsArray)-1 {
					currTab++
					tabsArray[currTab].Select(0)
				}
			})
			ui.SetKeybinding("Up", func() {
				if tabsArray[currTab].Selected() > 0 {
					tabsArray[currTab].Select(tabsArray[currTab].Selected() - 1)
					// Scroll if height is greater than hint height
					if tabsArray[currTab].Length() > scrollsArray[currTab].Size().Y {
						scrollsArray[currTab].Scroll(0, -1)
					}
				}
			})
			ui.SetKeybinding("Down", func() {
				if tabsArray[currTab].Selected() < tabsArray[currTab].Length()-1 {
					tabsArray[currTab].Select(tabsArray[currTab].Selected() + 1)
					if tabsArray[currTab].Length() > scrollsArray[currTab].Size().Y {
						scrollsArray[currTab].Scroll(0, 1)
					}
				}
			})

			// Select first database element, if any
			if databaseBar.Length() > 0 {
				databaseBar.Select(0)
			}

			if err := ui.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&file, "file", "f", "", "Path to plaintext file containing a MongoDB URI")
	cmd.PersistentFlags().StringVarP(&uri, "uri", "u", "", "A MongoDB URI")

	cmd.Execute()
}
