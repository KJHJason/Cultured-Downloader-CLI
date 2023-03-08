package main

import (
	"fmt"

	"github.com/KJHJason/Cultured-Downloader-CLI/cmds"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func main() {
	err := cmds.RootCmd.Execute()
	if err != nil {
		panic(
			fmt.Errorf(
				"error %d: unable to execute root command, more info => %v",
				utils.DEV_ERROR,
				err,
			),
		)
	}
}
