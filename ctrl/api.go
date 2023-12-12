package main

import "github.com/gin-gonic/gin"

var (
	downChannel = make(chan [3]string, 1)
)

func down(c *gin.Context) {
	name := c.PostForm("file_name")
	url := c.PostForm("url")
	path := c.PostForm("file_path")

	//呈递给下载器
	var info [3]string
	info[0], info[1], info[2] = name, url, path
	downChannel <- info
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
