package main

import (
	// "to-anki/utils"
	// "fmt"
	// "path/filepath"
	// "strings"
	"log"
	"to-anki/mubu"
	"to-anki/utils"
	// "github.com/beevik/etree"
)

func main() {
	config := utils.Config{}
	err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}
	anki := utils.AnkiApi{Config: config}
	mc := mubu.MubuConvert{Anki: anki, Config: config}
	mc.Do_chain()
}
