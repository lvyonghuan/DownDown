package engine

import "DownDown/config"

// InitEngine 初始化下载引擎
func InitEngine(config config.Config) *Engine {
	engine := new(Engine)
	engine.Config = config
	engine.DownFileNum = 0
	engine.downFileInfos = make(map[string]*DownFileInfo)

	//初始化限速器
	//将限速由chunkSize转化为令牌桶令牌size
	engine.downLimit = make(chan struct{}, config.DownLimit/config.ChunkSize)
	//填装令牌桶
	for i := 0; i < config.DownLimit/config.ChunkSize; i++ {
		engine.downLimit <- struct{}{}
	}

	return engine
}

// InitDownFileInfo 初始化下载文件对象
func (engine *Engine) InitDownFileInfo(fileName string, filePath string, url string) *DownFileInfo {
	downFileInfo := new(DownFileInfo)
	downFileInfo.FileName = fileName
	downFileInfo.FilePath = filePath
	downFileInfo.url = url
	downFileInfo.engine = engine

	//初始化管道
	downFileInfo.downManager.reDown = make(chan chunk, 1)
	downFileInfo.downManager.stop = make(chan struct{}, 1)

	return downFileInfo
}
