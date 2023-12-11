package engine

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// 获取要下载的文件的信息
func (downFileInfo *DownFileInfo) getFileInfo() (err error) {
	req, err := http.NewRequest("HEAD", downFileInfo.url, nil)
	if err != nil {
		return
	}

	client, err := Client(downFileInfo.engine.Config.Proxy)
	if err != nil {
		log.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//获取header
	header := resp.Header

	//获取文件大小
	fileSize := header.Get("Content-Length")

	downFileInfo.FileSize, err = strconv.ParseInt(fileSize, 10, 64)

	return err
}

// 根据下载文件长度进行分片操作
func (downFileInfo *DownFileInfo) chunker() (err error) {
	//获取分片大小
	chunkSize := downFileInfo.engine.Config.ChunkSize

	// 计算分片数量
	chunkNum := downFileInfo.FileSize / int64(chunkSize)

	// 计算最后一个分片的大小
	lastChunkSizeTemp := downFileInfo.FileSize % int64(chunkSize)
	//将其转化为int类型
	lastChunkSize := int(lastChunkSizeTemp)

	// 如果最后一个分片大小不为0，则分片数量加1
	if lastChunkSize != 0 {
		chunkNum++
	}

	//向下载管理器写入元信息
	downFileInfo.downManager.chunkNum = int(chunkNum) //分片数量
	downFileInfo.downManager.chunkSize = chunkSize    //分片大小

	//初始化下载队列
	downFileInfo.downManager.chunks = make(chan chunk, chunkNum)

	//创建分片
	//分片ID为分片顺序索引，从0开始
	//顺便计算range
	for i := 0; i < int(chunkNum); i++ {
		var chunk chunk
		chunk.chunkID = i
		chunk.chunkSize = chunkSize

		//计算range
		start := i * chunkSize
		chunk.start = start
		if i < int(chunkNum)-1 {
			end := start + chunkSize - 1
			chunk.end = end
			chunk.rangeSize = "bytes=" + strconv.FormatInt(int64(start), 10) + "-" + strconv.FormatInt(int64(end), 10)
		} else {
			end := start + lastChunkSize - 1
			chunk.end = end
			chunk.rangeSize = "bytes=" + strconv.FormatInt(int64(start), 10) + "-" + strconv.FormatInt(int64(end), 10)
		}

		//waitGroup+1
		downFileInfo.downManager.waitGroup.Add(1)

		downFileInfo.downManager.chunks <- chunk
	}

	return nil
}

// 根据分片创建下载任务
func (downFileInfo *DownFileInfo) createTask() (err error) {
	//创建下载任务
	//申请一块磁盘空间
	file, err := os.Create(downFileInfo.FilePath + "/" + downFileInfo.FileName)
	if err != nil {
		return err
	}
	defer file.Close()
	downFileInfo.downManager.file = file
	err = file.Truncate(downFileInfo.FileSize)
	if err != nil {
		return err
	}

	//建立结束监听器
	go downFileInfo.stop()
	log.Println("下载任务" + downFileInfo.FileName + "创建成功")
	log.Println("开始下载文件" + downFileInfo.FileName + "...")

	//开始下载
	for downFileInfo.downManager.downChunk != downFileInfo.downManager.chunkNum {
		select {
		case chunk := <-downFileInfo.downManager.chunks:
			go downFileInfo.downChunk(chunk)
		case chunk := <-downFileInfo.downManager.reDown:
			downFileInfo.downManager.chunks <- chunk //重新加入到队列中
		case <-downFileInfo.downManager.stop:
			continue
		}
	}

	log.Println("文件" + downFileInfo.FileName + "下载完成")

	return err
}

// 下载指定分片(并发)
// 如果报错,则将分片重新加入到下载队列当中去
func (downFileInfo *DownFileInfo) downChunk(chunk chunk) {
	//构造请求头
	req, err := http.NewRequest("GET", downFileInfo.url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Range", chunk.rangeSize)

	//设置代理
	client, err := Client(downFileInfo.engine.Config.Proxy)
	if err != nil {
		log.Println(err)
	}

	//发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		downFileInfo.downManager.reDown <- chunk
		return
	}
	defer resp.Body.Close()

	//读取body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		downFileInfo.downManager.reDown <- chunk
		return
	}

	//创建分片文件
	err = downFileInfo.writeChunk(chunk, body)
	if err != nil {
		log.Println(err)
		downFileInfo.downManager.reDown <- chunk
		return
	}

	//waitGroup-1
	downFileInfo.downManager.waitGroup.Done()
	downFileInfo.downManager.mu.Lock()
	downFileInfo.downManager.downChunk++
	downFileInfo.downManager.mu.Unlock()
	log.Println("下载进度：" + strconv.Itoa(downFileInfo.downManager.downChunk) + "/" + strconv.Itoa(downFileInfo.downManager.chunkNum))
}

// 利用偏移量将分片写入到文件中
func (downFileInfo *DownFileInfo) writeChunk(chunk chunk, data []byte) (err error) {
	offSet := int64(chunk.chunkID) * int64(chunk.chunkSize)
	//log.Println("序号:", chunk.chunkID, " 偏移量：", offSet, " 大小：", len(data), " 起始：", chunk.start, " 结束：", chunk.end, " range：", chunk.rangeSize)
	_, err = downFileInfo.downManager.file.WriteAt(data, offSet)
	return err
}

// 结束下载监听器
func (downFileInfo *DownFileInfo) stop() {
	downFileInfo.downManager.waitGroup.Wait()
	downFileInfo.downManager.stop <- struct{}{}
	close(downFileInfo.downManager.chunks)
	close(downFileInfo.downManager.stop)
	close(downFileInfo.downManager.reDown)
}
