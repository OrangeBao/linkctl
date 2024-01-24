package main

import (
	"context"
	"fmt"

	"github.com/kosmos.io/linkctl/cmd/floater/app"
)

func main() {
	ctx := context.TODO()
	cmd := app.NewFloaterCommand(ctx)
	err := cmd.Execute()
	if err != nil {
		fmt.Print(err)
	}
}
