package validation

import (
	openai "GEEK_back/client/openAI"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
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

type Choice struct {
    Message Message `json:"message"`
}

type ChatCompletionResponse struct {
    Choices []Choice `json:"choices"`
}

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
}

func (g *GigaChat) ValidAnswer(token, userAnswer, rightAnswer, question string, history []*openai.Message) ([]Choice, error) {
	url := g.BaseURL+"/chat/completions"
	
	systemPromt := `"\nТы - эксперт, который проверяет правильность \
	решения теста. Тест является не обычным. На вопросы теста пользователь должен \
	отвечать с помощью LLM, которая интегрирована в приложение. Вопросы подобраны \
	специально так, чтобы пользователь не мог просто скопировать вопрос и отправить его \
	в модель, а затем получить ответ. При таком подходе LLM не сможет правильно ответить \
	на вопрос, поэтому пользователь должен показать все свои умения в промт-инжиниринге. \
	Твоя задача заключается в том, чтобы проверить насколько пользователь ответил правильно. \
	Ты получишь сам вопрос из теста, эталонный ответ на него, ответ пользователя, а так же \
	историю диалога пользователя с LLM. Ответ пользователя необязательно должен точь в точь \
	совпадать с эталонным ответом, поэтому ты должен оценить насколько пользователь близок \
	к правильному ответу. Так же не забывай оценить ещё и историю диалога \
	пользователя с моделью. Если диалог с ней никак не связан с решением вопроса или если диалог \
	пуст, то отнимай за это баллы. Верни ответ в формате 10-бальной оценки. \
	Больше ничего писать в ответе не нужно.`

	userPromt := fmt.Sprintf(`
		"\nВопрос: %s\nЭталонный ответ: %s\nОтвет пользователя: %s\ 
		\nИстория диалога с LLM: %v"`,
		 question, rightAnswer, userAnswer, history)

	var modelResponse ChatCompletionResponse

	body := map[string]interface{}{
		"model": "GigaChat-2-Max",
		"messages": []map[string]string{
			{
				"role": "system",
				"content": fmt.Sprintf("System Promt: %s", systemPromt),
			},
			{
				"role": "user",
				"content": fmt.Sprintf("User Promt: %s", userPromt),
			},
		},
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&modelResponse); err != nil {
		return nil, err
	}

	return modelResponse.Choices, nil
}