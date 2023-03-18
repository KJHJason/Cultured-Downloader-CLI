package pixivmobile

import (
	"fmt"
	"path/filepath"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Process the artwork JSON and returns a slice of map that contains the urls of the images and the file path
func (pixiv *PixivMobile) processArtworkJson(artworkJson *models.PixivMobileIllustJson, downloadPath string) ([]*request.ToDownload, *models.Ugoira, error) {
	if artworkJson == nil {
		return nil, nil, nil
	}

	artworkId := fmt.Sprintf("%d", int64(artworkJson.Id))
	artworkTitle := artworkJson.Title
	artworkType := artworkJson.Type
	illustratorName := artworkJson.User.Name
	artworkFolderPath := utils.GetPostFolder(
		filepath.Join(downloadPath, utils.PIXIV_TITLE), illustratorName, artworkId, artworkTitle,
	)

	if artworkType == "ugoira" {
		ugoiraInfo, err := pixiv.getUgoiraMetadata(artworkId, artworkFolderPath)
		if err != nil {
			return nil, nil, err
		}
		return nil, ugoiraInfo, nil
	}

	var artworksToDownload []*request.ToDownload
	singlePageImageUrl := artworkJson.MetaSinglePage.OriginalImageUrl
	if singlePageImageUrl != "" {
		artworksToDownload = append(artworksToDownload, &request.ToDownload{
			Url:      singlePageImageUrl,
			FilePath: artworkFolderPath,
		})
	} else {
		for _, image := range artworkJson.MetaPages {
			imageUrl := image.ImageUrls.Original
			artworksToDownload = append(artworksToDownload, &request.ToDownload{
				Url:      imageUrl,
				FilePath: artworkFolderPath,
			})
		}
	}
	return artworksToDownload, nil, nil
}

// The same as the processArtworkJson function but for mutliple JSONs at once
// (Those with the "illusts" key which holds a slice of maps containing the artwork JSON)
func (pixiv *PixivMobile) processMultipleArtworkJson(resJson *models.PixivMobileArtworksJson, downloadPath string) ([]*request.ToDownload, []*models.Ugoira, []error) {
	if resJson == nil {
		return nil, nil, nil
	}

	artworksMaps := resJson.Illusts
	if len(artworksMaps) == 0 {
		return nil, nil, nil
	}

	var errSlice []error
	var ugoiraToDl []*models.Ugoira
	var artworksToDl []*request.ToDownload
	for _, artwork := range artworksMaps {
		artworks, ugoira, err := pixiv.processArtworkJson(artwork, downloadPath)
		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}
		if ugoira != nil {
			ugoiraToDl = append(ugoiraToDl, ugoira)
			continue
		}
		artworksToDl = append(artworksToDl, artworks...)
	}
	return artworksToDl, ugoiraToDl, errSlice
}
