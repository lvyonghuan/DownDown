package main

import (
	config2 "DownDown/config"
	"DownDown/engine"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	config, err := config2.ReadConfig()
	if err != nil {
		panic(err)
	}

	e := engine.InitEngine(config)
	err = scanResume(e)
	if err != nil {
		panic(err)
	}

	go downFile(e)
	listenDownLoadRequest()
}

func scanResume(engine *engine.Engine) error {
	err := engine.InitIndexFile()
	if err != nil {
		return err
	}

	err = engine.ScanResume()

	return err
}

func listenDownLoadRequest() {
	r := gin.Default()

	r.GET("/down", down)

	r.Run(":8080")
}

func downFile(engine *engine.Engine) {
	for {
		select {
		case info := <-downChannel:
			err := engine.DownLoadFile(info[0], info[1], info[2])
			if err != nil {
				log.Println(err)
			}
		}
	}
}
