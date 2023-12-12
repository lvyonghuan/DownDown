package engine

import (
	"DownDown/util"
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

//断点续传

// InitIndexFile 初始化索引文件
func (engine *Engine) InitIndexFile() error {
	file, err := os.OpenFile(util.ResumeIndexPath, os.O_CREATE, 0777)
	engine.resumeIndex = file
	return err
}

// 添加文件索引
func (downFileInfo *DownFileInfo) addIndexFile() error {
	//如果是断点续传，则不需要添加索引
	if downFileInfo.isResume {
		return nil
	}

	downFileInfo.engine.resumeMu.Lock()
	//在索引文件中加入该任务
	_, err := downFileInfo.engine.resumeIndex.WriteString(downFileInfo.FileName + ".txt\n")
	if err != nil {
		return err
	}
	downFileInfo.engine.resumeMu.Unlock()

	//初始化resume内容
	//第一行：FileSize|FilePath|url
	//第二行：chunkNum|chunkSize
	//第三行：已经下载好的切片ID。格式：
	//ID|ID|ID|......|ID
	var lines []string
	lines = append(lines, strconv.FormatInt(downFileInfo.FileSize, 10)+util.ResumeDelimiter+downFileInfo.FilePath+util.ResumeDelimiter+downFileInfo.url)
	lines = append(lines, strconv.Itoa(downFileInfo.downManager.chunkNum)+util.ResumeDelimiter+strconv.Itoa(downFileInfo.downManager.chunkSize))
	lines = append(lines, "")
	output := strings.Join(lines, "\n")

	//创建文件resume
	file, err := os.OpenFile(util.ResumeIndexDir+downFileInfo.FileName+".txt", os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	//写入内容
	_, err = file.WriteString(output)

	downFileInfo.downManager.resumeFile = file
	return err
}

// 在任务完成之后，删除resume索引及其文件
func (downFileInfo *DownFileInfo) deleteIndexFile() error {
	downFileInfo.engine.resumeMu.Lock()
	// 在任务完成之后，删除resume索引及其文件
	//删除索引文件中的索引
	//创建带缓冲区的文件读取器
	scanner := bufio.NewScanner(downFileInfo.engine.resumeIndex)

	// 逐行读取文件到一个数组中
	var lines []string
	fileName := downFileInfo.FileName + ".txt"
	for scanner.Scan() {
		line := scanner.Text()
		if line != fileName {
			lines = append(lines, line)
		}
	}

	// 重置文件读写位置
	downFileInfo.engine.resumeIndex.Seek(0, io.SeekStart)

	// 清空文件内容
	err := downFileInfo.engine.resumeIndex.Truncate(0)
	if err != nil {
		return err
	}

	// 将剩余的行写回文件
	for _, line := range lines {
		_, err := downFileInfo.engine.resumeIndex.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	downFileInfo.engine.resumeMu.Unlock()

	//删除索引文件
	downFileInfo.downManager.resumeFile.Close()
	time.Sleep(1 * time.Second)
	err = os.Remove(util.ResumeIndexDir + downFileInfo.FileName + ".txt")
	if err != nil {
		return err
	}

	return nil
}

// 按行读取文件对象索引，返回字符串切片，用于遍历重建对象
func (engine *Engine) readIndexFile() (err error) {
	var lines []string
	//获取索引文件对象
	file := engine.resumeIndex

	//创建带缓冲区的文件读取器
	scanner := bufio.NewScanner(file)

	//逐行读取文件
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	//检查扫描错误
	err = scanner.Err()

	engine.resumeList = lines

	return err
}

// 扫描文件夹，获取索引文件
func (engine *Engine) scanIndexDir() error {
	var resumeIndexes = engine.resumeList
	for _, index := range resumeIndexes {
		file, err := os.OpenFile(util.ResumeIndexDir+index, os.O_RDONLY, 0666)
		if err != nil {
			log.Println(err)
			continue
		}

		// 重建下载对象
		fileInfo, err := scanIndexFile(file)
		if err != nil {
			log.Println(err)
			continue
		}
		fileInfo.FileName = index[:len(index)-4]
		fileInfo.isResume = true

		//重建chan
		fileInfo.downManager.stop = make(chan struct{}, 1)
		fileInfo.downManager.reDown = make(chan chunk, 1)

		//重建与engine的关系
		fileInfo.engine = engine

		//加入到下载队列
		engine.downFileInfos[fileInfo.FileName] = fileInfo
	}
	return nil
}

// 扫描索引文件，重建建立对象
// 索引文件格式：
// 文件名.txt
// 第一行：FileSize|FilePath|url
// 第二行：chunkNum|chunkSize
// 第三行：已经下载好的切片ID。格式：
// ID|ID|ID|......|ID
func scanIndexFile(resumeFile *os.File) (fileInfo *DownFileInfo, err error) {
	//初始化文件对象
	fileInfo = new(DownFileInfo)

	//创建带缓冲区的文件读取器
	scanner := bufio.NewScanner(resumeFile)

	//读取第一行
	scanner.Scan()
	line := scanner.Text()
	parts := strings.Split(line, util.ResumeDelimiter)
	fileInfo.FileSize, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	fileInfo.FilePath, fileInfo.url = parts[1], parts[2]

	//读取第二行
	scanner.Scan()
	line = scanner.Text()
	parts = strings.Split(line, util.ResumeDelimiter)
	fileInfo.downManager.chunkNum, err = strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}
	fileInfo.downManager.chunkSize, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	//重新初始化下载队列
	var chunks []chunk
	fileInfo.downManager.chunks = make(chan chunk, fileInfo.downManager.chunkNum)
	for i := 0; i < fileInfo.downManager.chunkNum; i++ {
		var chunk chunk
		chunk.chunkID = i
		chunk.chunkSize = fileInfo.downManager.chunkSize

		//计算range
		start := i * fileInfo.downManager.chunkSize
		chunk.start = start
		if i < fileInfo.downManager.chunkNum-1 {
			end := start + fileInfo.downManager.chunkSize - 1
			chunk.end = end
			chunk.rangeSize = "bytes=" + strconv.FormatInt(int64(start), 10) + "-" + strconv.FormatInt(int64(end), 10)
		} else {
			end := start + int(fileInfo.FileSize%int64(fileInfo.downManager.chunkSize)) - 1
			chunk.end = end
			chunk.rangeSize = "bytes=" + strconv.FormatInt(int64(start), 10) + "-" + strconv.FormatInt(int64(end), 10)
		}
		chunks = append(chunks, chunk)
	}

	//读取第三行
	scanner.Scan()
	line = scanner.Text()
	parts = strings.Split(line, util.ResumeDelimiter)
	for _, part := range parts {
		if part == "" {
			break
		}
		chunkID, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		fileInfo.downManager.downChunk++
		chunks[chunkID].isDown = true
	}

	//将未下载的分片加入下载队列
	for _, chunk := range chunks {
		if chunk.isDown == false {
			fileInfo.downManager.waitGroup.Add(1)
			fileInfo.downManager.chunks <- chunk
		}
	}

	fileInfo.downManager.resumeFile = resumeFile

	return fileInfo, err
}

// 向resume文件中写入已经下载好的分片ID
func (downFileInfo *DownFileInfo) writeResumeFile(ID int) error {
	file := downFileInfo.downManager.resumeFile
	downFileInfo.downManager.reMu.Lock()
	defer downFileInfo.downManager.reMu.Unlock()

	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	lines, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	//按行分割
	linesArray := strings.Split(string(lines), "\n")
	linesArray[2] += strconv.Itoa(ID) + "|"

	// 将修改后的内容写回文件
	output := strings.Join(linesArray, "\n")
	err = os.WriteFile(util.ResumeIndexDir+downFileInfo.FileName+".txt", []byte(output), 0666)
	return err
}
