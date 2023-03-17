package main

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/cmds"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func main() {
	err := utils.DeleteEmptyAndOldLogs()
	if err != nil {
		utils.LogError(err, "", false, utils.ERROR)
	}
	cmds.RootCmd.Execute()
}
