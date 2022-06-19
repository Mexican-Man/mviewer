package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marcusolsson/tui-go"
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
			if err != nil {
				return fmt.Errorf("failed to connect to mongodb: %v", err)
			}

			// Read databases from db
			dbs, err := client.ListDatabaseNames(ctx, bson.D{})
			if err != nil {
				return fmt.Errorf("failed to list databases: %v", err)
			}
			fmt.Println(dbs)

			// Read collects in db
			collections, err := client.Database("test").ListCollections(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to list collections: %v", err)
			}
			fmt.Println(collections)

			// Add sidebar
			sidebar := tui.NewVBox()

			// Add databases
			sidebar.Append(tui.NewLabel("DATABSES"))
			for _, db := range dbs {
				sidebar.Append(tui.NewLabel(db))
			}
			sidebar.SetBorder(true)

			history := tui.NewVBox()

			// for _, m := range dbs {
			// 	history.Append(tui.NewHBox(
			// 		tui.NewLabel(m.time),
			// 		tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", m.username))),
			// 		tui.NewLabel(m.message),
			// 		tui.NewSpacer(),
			// 	))
			// }

			historyScroll := tui.NewScrollArea(history)
			historyScroll.SetAutoscrollToBottom(true)

			historyBox := tui.NewVBox(historyScroll)
			historyBox.SetBorder(true)

			input := tui.NewEntry()
			input.SetFocused(true)
			input.SetSizePolicy(tui.Expanding, tui.Maximum)

			inputBox := tui.NewHBox(input)
			inputBox.SetBorder(true)
			inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

			chat := tui.NewVBox(historyBox, inputBox)
			chat.SetSizePolicy(tui.Expanding, tui.Expanding)

			input.OnSubmit(func(e *tui.Entry) {
				history.Append(tui.NewHBox(
					tui.NewLabel(time.Now().Format("15:04")),
					tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", "john"))),
					tui.NewLabel(e.Text()),
					tui.NewSpacer(),
				))
				input.SetText("")
			})

			root := tui.NewHBox(sidebar, chat)

			ui, err := tui.New(root)
			if err != nil {
				log.Fatal(err)
			}
			ui.SetKeybinding("Esc", func() { ui.Quit() })

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
