package engine

import "log"

// DownControl 下载调度器
func (engine *Engine) DownControl() {
	for {
		select {
		case downInfo := <-engine.DownInfo:
			go engine.DownLoadFile(downInfo[0], downInfo[1], downInfo[2])
		}
	}
}

func (engine *Engine) DownLoadFile(name, url, path string) {
	// 初始化下载对象
	downFileInfo := engine.InitDownFileInfo(name, path, url)

	// 获取下载文件信息
	log.Println("获取文件" + name + "信息中...")
	err := downFileInfo.getFileInfo()
	if err != nil {
		log.Println("下载文件"+name+"失败", "错误信息：", err)
	}

	//分片
	err = downFileInfo.chunker()
	if err != nil {
		log.Println("下载文件"+name+"失败", "错误信息：", err)
	}

	// 创建下载任务
	log.Println("创建下载任务" + name + "中...")
	err = downFileInfo.createTask()
	if err != nil {
		log.Println("下载文件"+name+"失败", "错误信息：", err)
	}
}

func (engine *Engine) ScanResume() error {
	err := engine.readIndexFile()
	if err != nil {
		return err
	}

	err = engine.scanIndexDir()
	if err != nil {
		return err
	}

	return nil
}

// ReDownResume 重新下载未完成下载的任务
func (engine *Engine) ReDownResume() {
	for _, fileName := range engine.resumeList {
		fileInfo := engine.downFileInfos[fileName[:len(fileName)-4]]
		err := fileInfo.createTask()
		if err != nil {
			log.Println(err)
			continue
		}
	}
}
