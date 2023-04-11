package main

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/cmds"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

func main() {
	request.CheckInternetConnection()

	if err := request.CheckVer(); err != nil {
		utils.LogError(err, "", false, utils.ERROR)
	}

	if err := utils.DeleteEmptyAndOldLogs(); err != nil {
		utils.LogError(err, "", false, utils.ERROR)
	}
	cmds.RootCmd.Execute()
}
