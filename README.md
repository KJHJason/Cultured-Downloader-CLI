<h1 align="center">
    <img src="https://raw.githubusercontent.com/KJHJason/Cultured-Downloader/main/res/cultured_downloader_logo.png" width="100px" height="100px" alt="Cultured Downloader Logo">
    <br>
    Cultured Downloader CLI
</h1>

<div align="center">
    <a href="https://github.com/KJHJason/Cultured-Downloader-CLI/releases">
        <img src="https://img.shields.io/github/v/release/KJHJason/Cultured-Downloader-CLI?include_prereleases&label=Latest%20Release">
    </a>
    <a href="https://github.com/KJHJason/Cultured-Downloader-CLI/issues">
        <img src="https://img.shields.io/github/issues/KJHJason/Cultured-Downloader-CLI">
    </a>
    <a href="https://github.com/KJHJason/Cultured-Downloader-CLI/pulls">
        <img src="https://img.shields.io/github/issues-pr/KJHJason/Cultured-Downloader-CLI">
    </a>
    <img src="https://img.shields.io/github/downloads/KJHJason/Cultured-Downloader-CLI/latest/total">
</div>

## Introduction

Based on the original [Cultured Downloader](https://github.com/KJHJason/Cultured-Downloader), this is a command line interface (CLI) version of the program with more flexibility for automating downloads from your favorite platforms.

You might have noticed that the CLI version of the program is coded in Go, while the original is coded in Python. 
This is because of its in-built concurrency and the fact that I wanted to learn Go.

In terms of performance regarding the CLI version and the original, the CLI version will be faster as it is coded in Go, which is a compiled language unlike Python, which is an interpreted language. Additionally, Go's uses [Goroutines](https://yourbasic.org/golang/goroutines-explained/) for its concurrency which is much efficient than other langauges which is why I picked Go for this project.

## OS Support

The CLI version of the program is currently only available for Windows, Linux and macOS (In theory).

This program has only been tested on Windows 10. Hence, if you encounter any issues on other operating systems, I may not be able to help you.

## Usage Example

The example below assumes you are using [Go](https://go.dev/dl/) to run the program.
Otherwise, instead of `go run cultured_downloader.go`, you can run the executable file by typing `./cultured_downloader.exe` for Windows or `./cultured_downloader` for Linux in the terminal.

Note:
- For boolean flags, you can use `-flag` or `-flag=true` to enable the flag, and `-flag=false` to disable the flag.

Help (Either one of the following will work):
```
go run cultured_downloader.go
go run cultured_downloader.go -help
```

Downloading from a Fantia Fanclub ID:
```
go run cultured_downloader.go -fantia_session <add yours here> -fanclub_id 123456
```

Downloading from a Pixiv Fanbox Post ID:
```
go run cultured_downloader.go -fanbox_session <add yours here> -fanbox_post 123456 -gdrive_api_key <add your api key>
```

Downloading from a Pixiv Artwork ID (that is a Ugoira):
```
go run cultured_downloader.go -pixiv_session <add yours here> -artwork_id 12345678 -ugoira_output_format ".gif" -delete_ugoira_zip=false
```

## Flags

```
  -artwork_id string
        Artwork ID(s) to download.
        For multiple IDs, separate them with a space.
        Example: "12345 67891"
  -artwork_type string
        Artwork Type Options:
        - illust_and_ugoira: Restrict downloads to illustrations and ugoira only
        - manga: Restrict downloads to manga only
        - all: Include both illustrations, ugoira, and manga artworks
        Note that you can only specify ONE artwork type per run! (default "all")
  -creator_id string
        Pixiv Fanbox Creator ID(s) to download from.
        For multiple IDs, separate them with a space.
        Example: "12345 67891"
  -delete_ugoira_zip
        Whether to delete the downloaded ugoira zip file after conversion. (default true)
  -download_attachments
        Whether to download the attachments of a post. (default true)
  -download_gdrive
        Whether to download the Google Drive links of a Pixiv Fanbox post. (default true)
  -download_images
        Whether to download the images of a post. (default true)
  -download_path string
        Configure the path to download the files to and save it for future runs.
        Note:
        If you had used the "-download_path" flag before or
        had used the Cultured Downloader software, you can leave this argument empty.
  -download_thumbnail
        Whether to download the thumbnail of a post. (default true)
  -fanbox_post string
        Pixiv Fanbox post ID(s) to download.
        For multiple IDs, separate them with a space.
        Example: "12345 67891"
  -fanbox_session string
        Your FANBOXSESSID cookie value to use for the requests to Pixiv Fanbox.
  -fanclub_id string
        Fantia Fanclub ID(s) to download from.
        For multiple IDs, separate them with a space.
        Example: "12345 67891"
  -fantia_post string
        Fantia post ID(s) to download.
        For multiple IDs, separate them with a space.
        Example: "12345 67891"
  -fantia_session string
        Your _session_id cookie value to use for the requests to Fantia.
  -ffmpeg_path string
        Configure the path to the FFmpeg executable.
        Download Link: https://ffmpeg.org/download.html
         (default "ffmpeg")
  -gdrive_api_key string
        Google Drive API key to use for downloading gdrive files.
        Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md
  -help
        Show the list of arguments with its description.
  -illustrator_id string
        Illustrator ID(s) to download.
        For multiple IDs, separate them with a space.
        Example: "12345 67891"
  -page_num string
        Min and max page numbers to search for corresponding to the order of the supplied tag names.
        Format: "pageNum" or "min-max"
        Example: "1" or "1-10"
  -pixiv_session string
        Your PHPSESSID cookie value to use for the requests to Pixiv.
  -rating_mode string
        Rating Mode Options:
        - r18: Restrict downloads to R-18 artworks
        - safe: Restrict downloads to all ages artworks
        - all: Include both R-18 and all ages artworks
        Note that you can only specify ONE rating mode per run!
         (default "all")
  -search_mode string
        Search Mode Options:
        - s_tag: Match any post with SIMILAR tag name
        - s_tag_full: Match any post with the SAME tag name
        - s_tc: Match any post related by its title or caption
        Note that you can only specify ONE search mode per run!
         (default "s_tag_full")
  -sort_order string
        Download Order Options: date, popular, popular_male, popular_female
        Additionally, you can add the "_d" suffix for a descending order.
        Example: "popular_d"
        Note:
        - Pixiv Premium is needed in order to search by popularity. Otherwise, Pixiv's API will default to "date_d".    
        - You can only specify ONE tag name per run!
         (default "date_d")
  -tag_name string
        Tag names to search for and download related artworks.
        For multiple tags, separate them with a comma.
        Example: "tag name 1, tagName2"
  -ugoira_output_format string
        Output format for the ugoira conversion using FFmpeg.
        Accepted Extensions: .gif, .apng, .webp, .webm, .mp4
         (default ".gif")
  -ugoira_quality int
        Configure the quality of the converted ugoira (Only for .mp4 and .webm).
        This argument will be used as the crf value for FFmpeg.
        The lower the value, the higher the quality.
        Accepted values:
        - mp4: 0-51
        - webm: 0-63

        For more information, see:
        - mp4: https://trac.ffmpeg.org/wiki/Encode/H.264#crf
        - webm: https://trac.ffmpeg.org/wiki/Encode/VP9#constantq
         (default 10)
  -version
        Display the current version of the Cultured Downloader CLI software.
```