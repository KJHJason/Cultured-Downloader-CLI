package pixivmobile

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
	"github.com/pkg/browser"
)

type accessTokenInfo struct {
	accessToken string    // The access token that will be used to communicate with the Pixiv's Mobile API
	expiresAt   time.Time // The time when the access token expires
}

// Perform a S256 transformation method on a byte array
func S256(bytes []byte) string {
	hash := sha256.Sum256(bytes)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

var pixivOauthCodeRegex = regexp.MustCompile(`^[\w-]{43}$`)

// Start the OAuth flow to get the refresh token
func (pixiv *PixivMobile) StartOauthFlow() error {
	// create a random 32 bytes that is cryptographically secure
	codeVerifierBytes := make([]byte, 32)
	_, err := cryptorand.Read(codeVerifierBytes)
	if err != nil {
		// should never happen but just in case
		err = fmt.Errorf(
			"pixiv mobile error %d: failed to generate random bytes, more info => %v",
			utils.DEV_ERROR,
			err,
		)
		return err
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(codeVerifierBytes)
	codeChallenge := S256([]byte(codeVerifier))

	loginParams := map[string]string{
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
		"client":                "pixiv-android",
	}

	loginUrl := pixiv.loginUrl + "?" + utils.ParamsToString(loginParams)
	err = browser.OpenURL(loginUrl)
	if err != nil {
		color.Red("Pixiv: Failed to open browser: " + err.Error())
		color.Red("Please open the following URL in your browser:")
		color.Red(loginUrl)
	} else {
		color.Green("Opened a new tab in your browser to\n" + loginUrl)
	}

	color.Yellow("If unsure, follow the guide below:")
	color.Yellow("https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/pixiv_oauth_guide.md\n")
	for {
		var code string
		fmt.Print(color.YellowString("Please enter the code you received from Pixiv: "))
		_, err := fmt.Scanln(&code)
		fmt.Println()
		if err != nil {
			color.Red("Failed to read inputted code: " + err.Error())
			continue
		}
		if !pixivOauthCodeRegex.MatchString(code) {
			color.Red("Invalid code format...")
			continue
		}

		res, err := request.CallRequestWithData(
			&request.RequestArgs{
				Url:         pixiv.authTokenUrl,
				Method:      "POST",
				Timeout:     pixiv.apiTimeout,
				CheckStatus: true,
				UserAgent:   "PixivAndroidApp/5.0.234 (Android 11; Pixel 5)",
			},
			map[string]string{
				"client_id":      pixiv.clientId,
				"client_secret":  pixiv.clientSecret,
				"code":           code,
				"code_verifier":  codeVerifier,
				"grant_type":     "authorization_code",
				"include_policy": "true",
				"redirect_uri":   pixiv.redirectUri,
			},
		)
		if err != nil {
			color.Red("Please check if the code you entered is correct.")
			continue
		}

		var oauthFlowJson models.PixivOauthFlowJson
		err = utils.LoadJsonFromResponse(res, &oauthFlowJson)
		if err != nil {
			color.Red(err.Error())
			continue
		}

		refreshToken := oauthFlowJson.RefreshToken
		color.Green("Your Pixiv Refresh Token: " + refreshToken)
		color.Yellow("Please save your refresh token somewhere SECURE and do NOT share it with anyone!")
		return nil
	}
}

// Refresh the access token
func (pixiv *PixivMobile) refreshAccessToken() error {
	pixiv.accessTokenMu.Lock()
	defer pixiv.accessTokenMu.Unlock()

	res, err := request.CallRequestWithData(
		&request.RequestArgs{
			Url:       pixiv.authTokenUrl,
			Method:    "POST",
			Timeout:   pixiv.apiTimeout,
			UserAgent: pixiv.userAgent,
		},
		map[string]string{
			"client_id":      pixiv.clientId,
			"client_secret":  pixiv.clientSecret,
			"grant_type":     "refresh_token",
			"include_policy": "true",
			"refresh_token":  pixiv.refreshToken,
		},
	)
	if err != nil || res.StatusCode != 200 {
		const errPrefix = "pixiv mobile error"
		if err == nil {
			res.Body.Close()
			err = fmt.Errorf(
				"%s %d: failed to refresh token due to %s response from Pixiv\n"+
					"Please check your refresh token and try again or use the \"-pixiv_start_oauth\" flag to get a new refresh token",
				errPrefix,
				utils.RESPONSE_ERROR,
				res.Status,
			)
		} else {
			err = fmt.Errorf(
				"%s %d: failed to refresh token due to %v\n"+
					"Please check your internet connection and try again",
				errPrefix,
				utils.CONNECTION_ERROR,
				err,
			)
		}
		return err
	}

	var oauthJson models.PixivOauthJson
	err = utils.LoadJsonFromResponse(res, &oauthJson)
	if err != nil {
		return err
	}

	expiresIn := oauthJson.ExpiresIn - 15 // usually 3600 but minus 15 seconds to be safe
	pixiv.accessTokenMap.accessToken = oauthJson.AccessToken
	pixiv.accessTokenMap.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return nil
}

// Reads the response JSON and checks if the access token has expired,
// if so, refreshes the access token for future requests.
//
// Returns a boolean indicating if the access token was refreshed.
func (pixiv *PixivMobile) refreshTokenIfReq() (bool, error) {
	if pixiv.accessTokenMap.accessToken != "" && pixiv.accessTokenMap.expiresAt.After(time.Now()) {
		return false, nil
	}

	err := pixiv.refreshAccessToken()
	if err != nil {
		return true, err
	}
	return true, nil
}
