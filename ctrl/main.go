package main

import (
	config2 "DownDown/config"
	"DownDown/engine"

	"github.com/gin-gonic/gin"
)

var e *engine.Engine

func main() {
	config, err := config2.ReadConfig()
	if err != nil {
		panic(err)
	}

	e = engine.InitEngine(config)
	err = scanResumeAndDown(e)
	if err != nil {
		panic(err)
	}

	go e.DownControl()
	listenDownLoadRequest()
}

func scanResumeAndDown(engine *engine.Engine) error {
	err := engine.InitIndexFile()
	if err != nil {
		return err
	}

	err = engine.ScanResume()
	if err != nil {
		return err
	}

	go engine.ReDownResume()

	return err
}

func listenDownLoadRequest() {
	r := gin.Default()

	r.GET("/down", down)

	r.Run(":8080")
}
