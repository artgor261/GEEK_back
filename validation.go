// package validation

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

const gigaURL string = "https://gigachat.devices.sberbank.ru/api/v1/"

type GigaChat struct {
	APIKey	string
	BaseURL	string
	HTTP	*http.Client
}

type TokenResponse struct {
	Token	string	`json:"access_token"`
	Expired	uint64	`json:"expires_at"`
}

type Message struct {
	Role	string	`json:"role"`
	Content	string 	`json:"content"`
}

type Choices struct {
	Messages	Message	`json:"message"`
}

// type Choices struct {
// 	Messages	[]*Message	`json:"message"`
// }

func NewGigaChat(apiKey string) *GigaChat {
	return &GigaChat{APIKey: apiKey, BaseURL: gigaURL, HTTP: &http.Client{}}
}

func (g *GigaChat) GetToken() (string, error) {
	var token TokenResponse

	data := url.Values{}
    data.Set("scope", "GIGACHAT_API_PERS")
	
	url := "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", uuid.New().String())
	req.Header.Set("Authorization", "Basic "+g.APIKey)

	g.HTTP.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true,}}

	resp, err := g.HTTP.Do(req)
    if err != nil {
        return "Error: ", err
    }
    defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "Error: ", err
	}

	return token.Token, nil

	// return fmt.Sprintf("Authorization header: '%s'\n", req.Header.Get("Authorization")), nil
}

func (g *GigaChat) ValidAnswer() ([]Choices, error) {
	url := g.BaseURL+"/chat/completions"
	bearerToken, _ := g.GetToken()
	// var modelResponse Choices

	var modelResponse struct {
		Choice	[]Choices	`json:"choices"`
	}
	
	reader := strings.NewReader(`{
	"model": "GigaChat-2-Max",
	"messages": [
		{
			"role": "user",
			"content": "Привет, кто тебя создал?"
		}
	]
	}`)
	
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+bearerToken)

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&modelResponse); err != nil {
		return nil, err
	}

	return modelResponse.Choice, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}
	
	giga := &GigaChat{
		APIKey: os.Getenv("GIGACHAT_API_KEY"),
		BaseURL: "https://gigachat.devices.sberbank.ru/api/v1/",
		HTTP: &http.Client{},
	}

	// token, err := giga.GetToken()
	// if err != nil {
	// 	fmt.Printf("Error: %v", err)
	// }

	fmt.Println(giga.ValidAnswer())
}