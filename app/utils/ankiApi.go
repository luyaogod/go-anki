package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// AnkiConnectAPI接口
type AnkiApi struct {
	Config
}

type Payload struct {
	Action  string      `json:"action"`
	Version int         `json:"version"`
	Params  interface{} `json:"params"`
}

// post基接口
func (anki *AnkiApi) basePost(payload Payload) (repBody string, err error) {
	jsonData, err := json.Marshal(payload)
	// log.Println(string(jsonData))
	if err != nil {
		return "", err
	}

	rep, err := http.Post(anki.AnkiConnectHost, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer rep.Body.Close()

	body, err := io.ReadAll(rep.Body)
	if err != nil {
		return "", err
	}
	// 测试输出
	log.Println(string(body))
	return string(body), nil
}

// 添加用于自动化的卡片模板.
func (anki *AnkiApi) AddAutoModel() error {

	type CardTemplate struct {
		Name  string `json:"Name"`
		Front string `json:"Front"`
		Back  string `json:"Back"`
	}

	type Params struct {
		ModelName     string         `json:"modelName"`
		InOrderFields []string       `json:"inOrderFields"`
		CSS           string         `json:"css"`
		IsCloze       bool           `json:"isCloze"`
		CardTemplates []CardTemplate `json:"cardTemplates"`
	}

	payload := Payload{
		Action:  "createModel",
		Version: 6,
		Params: Params{
			ModelName:     anki.AutoModelName,
			InOrderFields: []string{"front", "back", "frontImg", "backImg"},
			CSS:           ".card {font-family: arial; font-size: 20px;text-align: center; color: black; background-color: white;}",
			IsCloze:       false,
			CardTemplates: []CardTemplate{
				{
					Name:  "卡片 1",
					Front: `{{front}}<div></div>{{frontImg}}`,
					Back:  `{{front}}<div>{{frontImg}}</div><hr id=answer>{{back}}<div>{{backImg}}</div>`,
				},
			},
		},
	}
	_, err := anki.basePost(payload)
	if err != nil {
		return err
	}
	return nil
}

// 创建单个卡片夹.
func (anki *AnkiApi) CreateDeck(deckPath string) error {
	type Params struct {
		Deck string `json:"deck"`
	}
	payload := Payload{
		Action:  "createDeck",
		Version: 6,
		Params:  Params{Deck: deckPath},
	}
	_, err := anki.basePost(payload)
	if err != nil {
		return err
	}
	return nil
}

// 创建多个卡片夹.
func (anki *AnkiApi) CreateDecks(deckPaths []string) error {
	for _, deckPath := range deckPaths {
		err := anki.CreateDeck(deckPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// Fields 表示笔记的字段
type Fields struct {
	Front string `json:"Front"`
	Back  string `json:"Back"`
}

// Media 媒体文件链接
type Media struct {
	URL      string   `json:"url"`
	Filename string   `json:"filename"`
	Fields   []string `json:"fields"`
}

// Note 表示一个笔记
type Note struct {
	DeckName  string   `json:"deckName"`
	ModelName string   `json:"modelName"`
	Fields    Fields   `json:"fields"`
	Tags      []string `json:"tags"`
	Picture   []Media  `json:"picture"`
}

// 返回新卡片的构造完成的参数
func (anki *AnkiApi) NewNote(
	path string,
	que string,
	ans string,
	queImgUrls []string,
	ansImgUrls []string,
) Note {
	files := Fields{
		Front: que,
		Back:  ans,
	}

	var mediaSlice []Media = make([]Media, 0)
	// 添加问题图片
	if len(queImgUrls) > 0 {
		for _, url := range queImgUrls {
			media := Media{
				URL:      url,
				Filename: url, // 未处理
				Fields:   []string{"frontImg"},
			}
			mediaSlice = append(mediaSlice, media)
		}
	}

	// 添加答案图片
	if len(ansImgUrls) > 0 {
		for _, url := range ansImgUrls {
			media := Media{
				URL:      url,
				Filename: url, // 未处理
				Fields:   []string{"backImg"},
			}
			mediaSlice = append(mediaSlice, media)
		}
	}

	note := Note{
		DeckName:  path,
		ModelName: anki.AutoModelName,
		Fields:    files,
		Tags:      []string{""},
		Picture:   mediaSlice,
	}
	return note
}

// 添加单个笔记或卡片
func (anki *AnkiApi) AddNote(note Note) error {
	type Params struct {
		Note Note `json:"note"`
	}
	params := Params{
		Note: note,
	}
	Payload := Payload{
		Action:  "addNote",
		Version: 6,
		Params:  params,
	}
	_, err := anki.basePost(Payload)
	if err != nil {
		return err
	}
	return nil
}

// 添加多个笔记或卡片
func (anki *AnkiApi) AddNotes(notes []Note) error {
	type Params struct {
		Notes []Note `json:"notes"`
	}

	params := Params{
		Notes: notes,
	}

	Payload := Payload{
		Action:  "addNotes",
		Version: 6,
		Params:  params,
	}

	_, err := anki.basePost(Payload)
	if err != nil {
		return err
	}
	return nil
}
