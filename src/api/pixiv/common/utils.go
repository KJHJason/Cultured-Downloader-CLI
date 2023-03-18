package pixivcommon

import "github.com/KJHJason/Cultured-Downloader-CLI/utils"

// Convert the page number to the offset as one page will have 60 illustrations.
//
// Usually for paginated results from Pixiv's mobile API, checkPixivMax should be set to true.
func ConvertPageNumToOffset(minPageNum, maxPageNum, perPage int, checkPixivMax bool) (int, int) {
	minOffset, maxOffset := utils.ConvertPageNumToOffset(
		minPageNum, 
		maxPageNum, 
		perPage,
	)
	if checkPixivMax {
		// Check if the offset is larger than Pixiv's max offset
		if maxOffset > 5000 {
			maxOffset = 5000
		}
		if minOffset > 5000 {
			minOffset = 5000
		}
	}
	return minOffset, maxOffset
}
