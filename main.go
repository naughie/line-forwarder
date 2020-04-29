package main

import (
    "log"
    "os"
    "encoding/json"
    "strings"
    "net/http"
    "net/url"
    "github.com/labstack/echo/v4"
)

type Token struct {
    AccessToken string `json:"access_token"`
    ExpiresIn int `json:"expires_in"`
    TokenType string `json:"token_type"`
}

func (s *Token) header() string {
    return s.TokenType + " " + s.AccessToken
}

type User struct {
    DisplayName string `json:"displayName"`
    UserID string `json:"userId"`
    Language string `json:"language"`
    PictureURL string `json:"pictureUrl"`
    StatusMessage string `json:"statusMessage"`
}

type LINEObject struct {
    Events []LINEEvent `json:"events"`
}

type LINEEvent struct {
    Source LINESource `json:"source"`
}

type LINESource struct {
    UserID string `json:"userId"`
}

func fetchAccessToken() (Token, error) {
    values := url.Values{}
    values.Add("grant_type", "client_credentials")
    values.Add("client_id", os.Getenv("LINE_CLIENT_ID"))
    values.Add("client_secret", os.Getenv("LINE_CLIENT_SECRET"))

    req, err := http.NewRequest("POST", "https://api.line.me/v2/oauth/accessToken", strings.NewReader(values.Encode()))

    if err != nil {
        return Token{}, err
    }

    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    res, err := client.Do(req)
    if err != nil {
        return Token{}, err
    }
    defer res.Body.Close()

    token := Token{}
    decoder := json.NewDecoder(res.Body)
    err = decoder.Decode(&token)
    return token, err
}

func getUser(id string, token Token) (User, error) {
    client := &http.Client{}
    req, err := http.NewRequest("GET", "https://api.line.me/v2/bot/profile/" + id, nil)
    if err != nil {
        return User{}, err
    }
    req.Header.Add("Authorization", token.header())
    res, err := client.Do(req)
    if err != nil {
        return User{}, err
    }
    defer res.Body.Close()

    user := User{}
    decoder := json.NewDecoder(res.Body)
    err = decoder.Decode(&user)
    return user, err
}

func sendToIFTTT(userName, botName string) error {
    values := url.Values{}
    values.Add("value1", userName)
    values.Add("value2", botName)

    req, err := http.NewRequest("POST", "https://maker.ifttt.com/trigger/line_message_received/with/key/casHCp6Yws_4_TWkgcEkpU", strings.NewReader(values.Encode()))

    if err != nil {
        return err
    }

    req.Header.Add("Content-Type", "application/json")

    client := &http.Client{}
    res, err := client.Do(req)
    if err != nil {
        return err
    }
    defer res.Body.Close()
    return nil
}

func forward(token Token, botName string) func(echo.Context) error {
    return func(c echo.Context) error {
        log.Println("Handling request ...")
        obj := LINEObject{}
        if err := c.Bind(&obj); err != nil {
            log.Println(err)
        }
        for _, e := range obj.Events {
            src := e.Source
            log.Println("    Source: ", src)
            user, err := getUser(src.UserID, token)
            if err != nil {
                log.Println(err)
            }
            log.Println("    DisplayName: ", user.DisplayName, ", BotName: ", botName)
            err = sendToIFTTT(user.DisplayName, botName)
            if err != nil {
                log.Println(err)
            }
        }
        log.Println("Done")
        return c.NoContent(http.StatusOK)
    }
}

func main() {
    token, err := fetchAccessToken()
    if err != nil {
        log.Println("Cannot fetch access token")
        return
    }

    e := echo.New()

    e.POST("/assistancedu", forward(token, "休校塾"))

    e.Logger.Fatal(e.Start(":8080"))
}
