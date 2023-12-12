package engine

import (
	config2 "DownDown/config"
	"testing"
)

func TestGetFileInfo(t *testing.T) {
	var config = config2.Config{
		Proxy:     "http://127.0.0.1:7890",
		ChunkSize: 10485760,
		DownLimit: 104857600,
	}
	engine := InitEngine(config)
	err := engine.DownLoadFile("facenet.pth", "https://huggingface.co/lllyasviel/Annotators/resolve/main/facenet.pth", "../test")
	if err != nil {
		t.Error(err)
	}
}
