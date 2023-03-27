<h1 align="center">
    <img src="https://raw.githubusercontent.com/KJHJason/Cultured-Downloader/main/res/cultured_downloader_logo.png" width="100px" height="100px" alt="Cultured Downloader Logo">
    <br>
    Cultured Downloader CLI
</h1>

<div align="center">
    <img src="https://img.shields.io/github/license/KJHJason/Cultured-Downloader-CLI" alt="license">
    <a href="https://github.com/KJHJason/Cultured-Downloader-CLI/releases">
        <img src="https://img.shields.io/github/v/release/KJHJason/Cultured-Downloader-CLI?include_prereleases&label=Latest%20Release" alt="latest release version">
    </a>
    <a href="https://github.com/KJHJason/Cultured-Downloader-CLI/issues">
        <img src="https://img.shields.io/github/issues/KJHJason/Cultured-Downloader-CLI" alt="issues">
    </a>
    <a href="https://github.com/KJHJason/Cultured-Downloader-CLI/pulls">
        <img src="https://img.shields.io/github/issues-pr/KJHJason/Cultured-Downloader-CLI" alt="pull requests">
    </a>
    <img src="https://img.shields.io/github/downloads/KJHJason/Cultured-Downloader-CLI/latest/total" alt="latest release downloads">
    <img src="https://img.shields.io/codeclimate/maintainability/KJHJason/Cultured-Downloader-CLI" alt="Code Climate maintainability">
</div>

## Introduction

Based on the original [Cultured Downloader](https://github.com/KJHJason/Cultured-Downloader), this is a command line interface (CLI) version of the program with more flexibility for automating downloads from your favourite platforms.

You might have noticed that the CLI version of the program is coded in Go, while the original is coded in Python. 
This is because of its in-built concurrency and the fact that I wanted to learn Go.

In terms of performance regarding the CLI version and the original, the CLI version will be faster as it is coded in Go, which is a compiled language unlike Python, which is an interpreted language. Additionally, Go's uses [Goroutines](https://yourbasic.org/golang/goroutines-explained/) for its concurrency which is much efficient than other langauges which is why I picked Go for this project.

## OS Support

The CLI version of the program is currently only available for Windows, Linux and macOS (In theory).

This program has only been tested on Windows 10. Hence, if you encounter any issues on other operating systems, I may not be able to help you.

## Known Issue(s)

### File access denied error when downloading with the `--overwrite=true` flag...

- What the flag does:
  - Regardless of this flag, the program will overwrite any incomplete downloaded file by verifying against the Content-Length header response.
    - GDrive downloads are not affected by this flag.
  - If the size of locally downloaded file does not match with the header response, the program will re-download the file.
  - Otherwise, if the Content-Length header is not present in the response, the program will only skip the file if the file already exists.
  - However, if there's a need to overwrite the skipped files, you can use this flag to do so like for Pixiv Fanbox which does not return a Content-Length header in their response header.
  - That said, the program will try to do a clean up of the incomplete downloads by deleting them as of version 1.1.1 and above when you forcefully terminate the program by pressing Ctrl+C so you should not need to use this flag anymore.
- Causes:
  - This is caused by the antivirus program on your PC, flagging the program as a malware and stopping it from executing normally.
  - This is due to how the programs works, it will try to check the image file size by doing a HEAD request first before doing a GET request for downloading.
    - In the event that the Content-Length is not in the response header, it will re-download the file if the overwrite flag is set to true.
    - Usually, this is not a problem for Fantia and Pixiv, but for Pixiv Fanbox, it will not return the Content-Length in the response header, which is why the program will try to re-download all the files and overwrite any existing files. Hence, the antivirus program will flag the program as a ransomware and stop it from executing normally.
- Solutions:
  - Avoid using the `--overwrite=true` flag and let the program do its clean up when you forcefully terminate the program by pressing Ctrl+C.
    - Files that failed to be deleted will be logged so you can manually delete them.
  - If you still need to use this flag, please exclude `cultured-downloader-cli.exe` or the compiled version of the program from your antivirus software as it can be flagged as a ransomware as previously explained above in the Causes.
    - `go run .` will also NOT work as it will still be blocked by the antivirus program. Hence, you will need to build the program first and then run it.
    - By running `go build . -o cultured-downloader-cli.exe` in the src directory of the project, it will build the program and create an executable file.

## Disclaimers

- Cultured Downloader is not liable for any damages caused.
- Pixiv:
  - Pixiv API calls are throttled to avoid being rate limited.
    - You can try using the `--refresh_token` flag instead of the `--session` flag if you prefer faster downloads as it generally has lesser API calls but at the expense of a less flexible download options.
  - Try not to overuse the program when downloading from Pixiv as it can cause your IP to be flagged by Cloudflare.

## Usage Example

The example below assumes you are using [Go](https://go.dev/dl/) to run the program.

Otherwise, instead of `go run . cultured_downloader.go`, you can run the executable file by typing `./cultured_downloader.exe` for Windows.

There is also compiled binaries for Linux and macOS in the [releases](https://github.com/KJHJason/Cultured-Downloader-CLI/releases) page.

Note:
- For flags that require a value, you can either use the `--flag_name value` format or the `--flag_name=value` format.
- For flags that allows multiple values, you can either use the `--flag_name value1,value2` format or the `--flag_name=value1,value2` format.
- Quotations like `--flag_name="value1,value2"` are not required but it is recommended to use them.
- For `--text_file="<path>"`, the contents of the text file should be separated by a new line as seen below,
```
  https://fantia.jp/posts/123123
  https://fantia.jp/fanclubs/1234
  https://fantia.jp/fanclubs/4321; 1-12
```
- You can add the `; <pageNum>` after the URL as well!
  - Leave empty if you want to download all pages or follow the format `1` to download page 1 or `1-10` to download pages 1 to 10.
    - In the example above, the second line will download all pages of the provided Fantia Fanclub URL while the third line will download pages 1 to 12 of the provided Fantia Fanclub URL.
  - Only for:
    - Fantia Fanclub URLs
    - Pixiv Fanbox Creator URLs
    - Pixiv Illustrator URLs
    - Pixiv Tag URLs
    - Kemono Party Creator URLs

Help:
```
go run . cultured_downloader.go -h
```

Downloading from multiple Fantia Fanclub IDs:
```
go run . cultured_downloader.go fantia --cookie_file="C:\Users\KJHJason\Desktop\fantia.jp_cookies.txt" --fanclub_id 123456,789123 --page_num 1,1-10 --dl_thumbnails=false
```

Downloading from a Pixiv Fanbox Post ID:
```
go run . cultured_downloader.go pixiv_fanbox --session="<add yours here>" --post_id 123456,789123 --gdrive_api_key="<add your api key>"
```

Downloading from a Pixiv Artwork ID (that is a Ugoira):
```
go run . cultured_downloader.go pixiv --session "<add yours here>" --artwork_id 12345678 --ugoira_output_format ".gif" --delete_ugoira_zip=false
```

Downloading from multiple Pixiv Artwork IDs:
```
go run . cultured_downloader.go pixiv --refresh_token="<add yours here>" --artwork_id 12345678,87654321
```

Downloading from Pixiv using a tag name:
```
go run . cultured_downloader.go pixiv --refresh_token="<add yours here>" --tag_name "tag1,tag2,tag3" --tag_page_num 1,4,2 --rating_mode safe --search_mode s_tag
```

## Base Flags

```
Cultured Downloader CLI is a command-line tool for downloading images, videos, etc. from various websites like Pixiv, Pixiv Fanbox, Fantia, and more.

Usage:
  cultured-downloader-cli [flags]
  cultured-downloader-cli [command]

Available Commands:
  fantia       Download from Fantia
  help         Help about any command
  kemono       Download from Kemono Party
  pixiv        Download from Pixiv
  pixiv_fanbox Download from Pixiv Fanbox

Flags:
      --download_path string   Configure the path to download the files to and save it for future runs.
                               Otherwise, the program will use the current working directory.
                               Note:
                               If you had used the "-download_path" flag before or
                               had used the Cultured Downloader Python program, the program will automatically use the path you had set.
  -h, --help                   help for cultured-downloader-cli
  -v, --version                version for cultured-downloader-cli

Use "cultured-downloader-cli [command] --help" for more information about a command.
```

## Fantia Flags

```
Supports downloads from Fantia Fanclubs and individual posts.

Usage:
  cultured-downloader-cli fantia [flags]

Flags:
  -c, --cookie_file string      Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.
                                You can generate a cookie file by using the "Get cookies.txt LOCALLY" extension for your browser.
                                Chrome Extension URL: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc
      --dl_attachments          Whether to download the attachments of a post on Fantia. (default true)
      --dl_gdrive               Whether to download the Google Drive links of a post on Fantia. (default true)
      --dl_images               Whether to download the images of a post on Fantia. (default true)
      --dl_thumbnails           Whether to download the thumbnail of a post on Fantia. (default true)
      --fanclub_id strings      Fantia Fanclub ID(s) to download from.
                                For multiple IDs, separate them with a comma.
                                Example: "12345,67891" (without the quotes)
      --gdrive_api_key string   Google Drive API key to use for downloading gdrive files.
                                Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md
  -h, --help                    help for fantia
      --log_urls                Log any detected URLs of the files that are being downloaded.
                                Note that not all URLs are logged, only URLs to external file hosting providers like MEGA, Google Drive, etc. are logged.
  -o, --overwrite               Overwrite any existing files if there is no Content-Length header in the response.
                                Usually used for Pixiv Fanbox when there are incomplete downloads.
      --page_num strings        Min and max page numbers to search for corresponding to the order of the supplied Fantia Fanclub ID(s).
                                Format: "num", "minNum-maxNum", or "" to download all pages
                                Leave blank to download all pages from each Fantia Fanclub.
      --post_id strings         Fantia post ID(s) to download.
                                For multiple IDs, separate them with a comma.
                                Example: "12345,67891" (without the quotes)
  -s, --session string          Your "_session_id" cookie value to use for the requests to Fantia.
      --text_file string        Path to a text file containing Fanclub and/or post URL(s) to download from Fantia.
      --user_agent string       Set a custom User-Agent header to use when communicating with the API(s) or when downloading.
```

## Pixiv Fanbox Flags

```
Supports downloads from Pixiv Fanbox creators and individual posts.

Usage:
  cultured-downloader-cli pixiv_fanbox [flags]

Flags:
  -c, --cookie_file string      Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.
                                You can generate a cookie file by using the "Get cookies.txt LOCALLY" extension for your browser.
                                Chrome Extension URL: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc
      --creator_id strings      Pixiv Fanbox Creator ID(s) to download from.
                                For multiple IDs, separate them with a comma.
                                Example: "12345,67891" (without the quotes)
      --dl_attachments          Whether to download the attachments of a Pixiv Fanbox post. (default true)
      --dl_gdrive               Whether to download the Google Drive links of a Pixiv Fanbox post. (default true)
      --dl_images               Whether to download the images of a Pixiv Fanbox post. (default true)
      --dl_thumbnails           Whether to download the thumbnail of a Pixiv Fanbox post. (default true)
      --gdrive_api_key string   Google Drive API key to use for downloading gdrive files.
                                Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md
  -h, --help                    help for pixiv_fanbox
      --log_urls                Log any detected URLs of the files that are being downloaded.
                                Note that not all URLs are logged, only URLs to external file hosting providers like MEGA, Google Drive, etc. are logged.
  -o, --overwrite               Overwrite any existing files if there is no Content-Length header in the response.
                                Usually used for Pixiv Fanbox when there are incomplete downloads.
      --page_num strings        Min and max page numbers to search for corresponding to the order of the supplied Pixiv Fanbox creator ID(s).
                                Format: "num", "minNum-maxNum", or "" to download all pages
                                Leave blank to download all pages from each creator.
      --post_id strings         Pixiv Fanbox post ID(s) to download.
                                For multiple IDs, separate them with a comma.
                                Example: "12345,67891" (without the quotes)
  -s, --session string          Your "FANBOXSESSID" cookie value to use for the requests to Pixiv Fanbox.
      --text_file string        Path to a text file containing creator and/or post URL(s) to download from Pixiv Fanbox.
      --user_agent string       Set a custom User-Agent header to use when communicating with the API(s) or when downloading.
```


## Pixiv Flags

```
Supports downloads from Pixiv by artwork ID, illustrator ID, tag name, and more.

Usage:
  cultured-downloader-cli pixiv [flags]

Flags:
      --artwork_id strings             Artwork ID(s) to download.
                                       For multiple IDs, separate them with a comma.
                                       Example: "12345,67891" (without the quotes)
      --artwork_type string            Artwork Type Options:
                                       - illust_and_ugoira: Restrict downloads to illustrations and ugoira only
                                       - manga: Restrict downloads to manga only
                                       - all: Include both illustrations, ugoira, and manga artworks
                                       Notes:
                                       - If you're using the "-pixiv_refresh_token" flag and are downloading by tag names, only "all" is supported. (default "all")
  -c, --cookie_file string             Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.
                                       You can generate a cookie file by using the "Get cookies.txt LOCALLY" extension for your browser.
                                       Chrome Extension URL: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc
      --delete_ugoira_zip              Whether to delete the downloaded ugoira zip file after conversion. (default true)
      --ffmpeg_path string             Configure the path to the FFmpeg executable.
                                       Download Link: https://ffmpeg.org/download.html (default "ffmpeg")
  -h, --help                           help for pixiv
      --illustrator_id strings         Illustrator ID(s) to download.
                                       For multiple IDs, separate them with a comma.
                                       Example: "12345,67891" (without the quotes)
      --illustrator_page_num strings   Min and max page numbers to search for corresponding to the order of the supplied illustrator ID(s).
                                       Format: "num", "minNum-maxNum", or "" to download all pages
                                       Leave blank to download all pages from each illustrator.
  -o, --overwrite                      Overwrite any existing files if there is no Content-Length header in the response.
                                       Usually used for Pixiv Fanbox when there are incomplete downloads.
      --rating_mode string             Rating Mode Options:
                                       - r18: Restrict downloads to R-18 artworks
                                       - safe: Restrict downloads to all ages artworks
                                       - all: Include both R-18 and all ages artworks
                                       Notes:
                                       - If you're using the "--refresh_token" flag, only "all" is supported. (default "all")
      --refresh_token string           Your Pixiv refresh token to use for the requests to Pixiv.
                                       If you're downloading from Pixiv, it is recommended to use this flag
                                       instead of the "--session" flag as there will be significantly lesser API calls to Pixiv.
                                       However, if you prefer more flexibility with your Pixiv downloads, you can use
                                       the "--session" flag instead at the expense of longer API call time due to Pixiv's rate limiting.
                                       Note that you can get your refresh token by running the program with the "--start_oauth" flag.
      --search_mode string             Search Mode Options:
                                       - s_tag: Match any post with SIMILAR tag name
                                       - s_tag_full: Match any post with the SAME tag name
                                       - s_tc: Match any post related by its title or caption (default "s_tag_full")
  -s, --session string                 Your "PHPSESSID" cookie value to use for the requests to Pixiv.
      --sort_order string              Download Order Options: date, popular, popular_male, popular_female
                                       Additionally, you can add the "_d" suffix for a descending order.
                                       Example: "popular_d"
                                       Note:
                                       - If using the "--refresh_token" flag, only "date", "date_d", "popular_d" are supported.
                                       - Pixiv Premium is needed in order to search by popularity. Otherwise, Pixiv's API will default to "date_d". (default "date_d")
      --start_oauth                    Whether to start the Pixiv OAuth process to get one's refresh token.
      --tag_name strings               Tag names to search for and download related artworks.
                                       For multiple tags, separate them with a comma.
                                       Example: "tag name 1, tagName2"
      --tag_page_num strings           Min and max page numbers to search for corresponding to the order of the supplied tag name(s).
                                       Format: "num", "minNum-maxNum", or "" to download all pages
                                       Leave blank to search all pages for each tag name.
      --text_file string               Path to a text file containing artwork, illustrator, and tag name URL(s) to download from Pixiv.
      --ugoira_output_format string    Output format for the ugoira conversion using FFmpeg.
                                       Accepted Extensions: .gif, .apng, .webp, .webm, .mp4
                                        (default ".gif")
      --ugoira_quality int             Configure the quality of the converted ugoira (Only for .mp4 and .webm).
                                       This argument will be used as the crf value for FFmpeg.
                                       The lower the value, the higher the quality.
                                       Accepted values:
                                       - mp4: 0-51
                                       - webm: 0-63
                                       For more information, see:
                                       - mp4: https://trac.ffmpeg.org/wiki/Encode/H.264#crf
                                       - webm: https://trac.ffmpeg.org/wiki/Encode/VP9#constantq (default 10)
      --user_agent string              Set a custom User-Agent header to use when communicating with the API(s) or when downloading.
```

## Kemono Party Flags

```
Supports downloads from creators and posts on Kemono Party.

Usage:
  cultured-downloader-cli kemono [flags]

Flags:
  -c, --cookie_file string      Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.
                                You can generate a cookie file by using the "Get cookies.txt LOCALLY" extension for your browser.
                                Chrome Extension URL: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc
      --creator_url strings     Kemono Party creator URL(s) to download from.
                                Multiple URLs can be supplied by separating them with a comma.
                                Example: "https://kemono.party/service/user/123,https://kemono.party/service/user/456" (without the quotes)
      --dl_attachments          Whether to download the attachments (images, zipped files, etc.) of a post on Kemono Party. (default true)
      --dl_gdrive               Whether to download the Google Drive links of a post on Kemono Party. (default true)
      --gdrive_api_key string   Google Drive API key to use for downloading gdrive files.
                                Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md
  -h, --help                    help for kemono
      --log_urls                Log any detected URLs of the files that are being downloaded.
                                Note that not all URLs are logged, only URLs to external file hosting providers like MEGA, Google Drive, etc. are logged.
  -o, --overwrite               Overwrite any existing files if there is no Content-Length header in the response.
                                Usually used for Pixiv Fanbox when there are incomplete downloads.
      --page_num strings        Min and max page numbers to search for corresponding to the order of the supplied Kemono Party creator URL(s).
                                Format: "num", "minNum-maxNum", or "" to download all pages
                                Leave blank to download all pages from each creator on Kemono Party.
      --post_url strings        Kemono Party post URL(s) to download.
                                Multiple URLs can be supplied by separating them with a comma.
                                Example: "https://kemono.party/service/user/123,https://kemono.party/service/user/456" (without the quotes)
  -s, --session string          Your Kemono Party "session" cookie value to use for the requests to Kemono Party.
                                Required to get pass Kemono Party's DDOS protection and to download from your favourites.
      --text_file string        Path to a text file containing creator and/or post URL(s) to download from Kemono Party.
      --user_agent string       Set a custom User-Agent header to use when communicating with the API(s) or when downloading.
```
