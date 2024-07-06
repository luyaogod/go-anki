package mubu

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"to-anki/utils"

	"github.com/beevik/etree"
)

type MubuConvert struct {
	Anki utils.AnkiApi
	utils.Config
}

func (mc *MubuConvert) ImgURLDecode(rawURL string) []string {
	if rawURL == "" {
		return []string{}
	}
	decodedURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		fmt.Println("Error decoding URL:", err)
		return []string{}
	}
	var urlList []map[string]interface{}
	err = json.Unmarshal([]byte(decodedURL), &urlList)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return []string{}
	}

	var result []string
	for _, u := range urlList {
		if uri, ok := u["uri"].(string); ok {
			result = append(result, mc.MubuBaseUrl+uri)
		} else {
			fmt.Println("Error: 'uri' is not a string")
		}
	}
	return result
}

func (mc *MubuConvert) TextJoin(text, note string) string {
	if text != "" && note != "" {
		return text + "，" + note
	} else if text != "" {
		return text
	} else if note != "" {
		return note
	}
	return ""
}

func (mc *MubuConvert) ReadOMPL(filePath string) (*etree.Document, error) {
	files, err := os.ReadDir(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件夹读取失败: %w", err)
	}

	var opmlFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".opml" {
			opmlFiles = append(opmlFiles, file.Name())
		}
	}

	if len(opmlFiles) == 0 {
		return nil, errors.New("指定路径未发现OPML文件")
	}

	if len(opmlFiles) > 1 {
		return nil, errors.New("指定路径OPML文件>1无法导入")
	}

	opmlFilePath := filepath.Join(filePath, opmlFiles[0])
	file, err := os.Open(opmlFilePath)
	if err != nil {
		return nil, fmt.Errorf("文件打开失败: %w", err)
	}
	defer file.Close()

	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(file); err != nil {
		return nil, fmt.Errorf("解析OPML失败: %w", err)
	}
	return doc, nil
}

type DecksAndCards struct {
	Decks []string
	Cards []utils.Note
}

// 解析OPML
func (mc *MubuConvert) ParseOPML(doc *etree.Document) (*DecksAndCards, error) {

	root := doc.SelectElement("opml")

	if root == nil {
		return nil, fmt.Errorf("文件格式错误!opml节点不存在")
	}

	title := root.FindElement(".//head/title")
	if title == nil {
		return nil, fmt.Errorf("文件格式错误!title节点不存在")
	}
	titleStr := title.Text()
	log.Printf("TITLE %v", titleStr)

	body := root.FindElement(".//body")
	if body == nil {
		return nil, fmt.Errorf("文件格式错误!body节点不存在")
	}

	outLineRoot := body.FindElement(".//outline")
	if outLineRoot == nil {
		return nil, fmt.Errorf("文件格式错误!根outline节点不存在")
	}

	// 卡片夹和卡片存储
	decks := make(map[string]string)
	var deckList []string
	var cards []utils.Note

	// 层序遍历辅助队列
	type node struct {
		parentNode *node
		data       *etree.Element
	}
	queue := []node{}

	// 根节点入队
	// 不指定路径
	if mc.SpecificPath == "" {
		// 不指定路径+部分导出
		// 处理方法：将outline作为根节点入队
		if titleStr == outLineRoot.SelectAttrValue("text", "") {
			node := node{
				parentNode: nil,
				data:       outLineRoot,
			}
			queue = append(queue, node)

			// 不指定路径+全部导出(常规情况)
			// 处理方法：将body作为根节点入队
		} else {
			body.CreateAttr("text", titleStr)
			node := node{
				parentNode: nil,
				data:       body,
			}
			queue = append(queue, node)
		}
		// 指定路径
	} else {
		// 指定路径+全部导出
		// 处理方法：按照不指定路径+全部导出处理
		if titleStr != outLineRoot.SelectAttrValue("text", "") {
			body.CreateAttr("text", titleStr)
			node := node{
				parentNode: nil,
				data:       body,
			}
			queue = append(queue, node)

			// 指定路径+部分导出(常规情况)
			// 处理方法：添加指定路径并将第一层outline作为根
		} else {
			deckList = append(deckList, mc.SpecificPath) //将指定路径添加到存储表中
			node := node{
				parentNode: nil,
				data:       outLineRoot,
			}
			queue = append(queue, node)
		}
	}

	//层序遍历
	for len(queue) > 0 {
		currentNode := queue[0]
		parentNode := currentNode.parentNode //当前节点的父节点
		queue = queue[1:]

		children := currentNode.data.FindElements("outline")
		//非叶节点，被视为卡片夹
		if len(children) > 0 {
			// 获取卡片夹名称
			currentName := currentNode.data.SelectAttrValue("text", "")

			// 构造卡片夹路径
			// 父结点为nil的是根节点
			// 根节点前面是否存在路径取决于SpecificPath是否指定路径
			if parentNode == nil {
				if mc.SpecificPath == "" {
					decks[currentName] = currentName
				} else {
					decks[currentName] = mc.SpecificPath + "::" + currentName
				}
				deckList = append(deckList, decks[currentName]) // 将deckaName添进deckNames保存

				// 不是叶子节点但是带有图片被视为问题叶子
				// 将与它的子节点合并为一个card
			} else if currentNode.data.SelectAttrValue("_mubu_images", "") != "" {
				card := mc.Anki.NewNote(
					decks[parentNode.data.SelectAttrValue("text", "")],
					mc.TextJoin(currentNode.data.SelectAttrValue("text", ""), currentNode.data.SelectAttrValue("_note", "")),
					mc.TextJoin(children[0].SelectAttrValue("text", ""), children[0].SelectAttrValue("_note", "")),
					mc.ImgURLDecode(currentNode.data.SelectAttrValue("_mubu_images", "")), //问题图片
					mc.ImgURLDecode(children[0].SelectAttrValue("_mubu_images", "")),      //答案图片
				)
				cards = append(cards, card)

				// 走到此处为普通卡片夹的处理
			} else {
				parentName := parentNode.data.SelectAttrValue("text", "")
				currentPath := decks[parentName] + "::" + currentName
				decks[currentName] = currentPath
				deckList = append(deckList, decks[currentName]) // 将deckaName添进deckNames保存
			}

			// 根据层序遍历将出队的当前节点的所有子节点入队
			for _, child := range children {
				node := node{
					parentNode: &currentNode,
					data:       child,
				}
				queue = append(queue, node)
			}

		} else {
			// 没有子节点的是叶子节点，叶子节点为问答卡片
			// 如果叶子子节点的父节点存在图片，则此处不构造卡片，由其父结点会与其共同构造卡片
			if parentNode.data.SelectAttrValue("_note", "") != "" || parentNode.data.SelectAttrValue("_mubu_images", "") != "" {
				continue
			}
			card := mc.Anki.NewNote(
				decks[parentNode.data.SelectAttrValue("text", "")],
				currentNode.data.SelectAttrValue("text", ""),
				currentNode.data.SelectAttrValue("_note", ""),
				[]string{}, //叶子卡片不应该存在问题图片
				mc.ImgURLDecode(currentNode.data.SelectAttrValue("_mubu_images", "")), //答案图片
			)
			cards = append(cards, card)

			// 叶子节点没有子节点无需将子节点入队
		}

	}

	return &DecksAndCards{deckList, cards}, nil
}

func (mc *MubuConvert) Do_chain() {
	log.Println("开始导入...")
	doc, err := mc.ReadOMPL(mc.InputFilePath)
	if err != nil {
		log.Panic(err)
	}
	decksAndCards, err := mc.ParseOPML(doc)
	if err != nil {
		log.Panic(err)
	}

	decks := decksAndCards.Decks
	cards := decksAndCards.Cards

	log.Println(&decks)

	// 创建卡片夹
	log.Println("开始导入...")
	log.Println("正在创建卡片夹...")
	err = mc.Anki.CreateDecks(decks)
	if err != nil {
		log.Panic(fmt.Errorf("decks创建失败 %v", err))
	}

	// 创建笔记模板
	log.Println("正在创建模板...")
	err = mc.Anki.AddAutoModel()
	if err != nil {
		log.Panic(fmt.Errorf("创建卡片模板失败 %v", err))
	}

	// 创建卡片
	// log.Println("正在导入卡片...")
	// for _, card := range cards {
	// 	err = mc.Anki.AddNote(card)
	// 	if err != nil {
	// 		log.Panic(fmt.Errorf("笔记添加失败 %v", err))
	// 	}
	// }

	// 创建卡片
	err = mc.Anki.AddNotes(cards)
	if err != nil {
		log.Panic(fmt.Errorf("笔记创建失败 %v", err))
	}

	log.Println("导入完成.")

	// // 序列化为JSON并打印
	// jsonData, err := json.MarshalIndent(decksAndCards, "", "  ")
	// if err != nil {
	// 	log.Panic(err)
	// }
	// log.Println(string(jsonData))
}
