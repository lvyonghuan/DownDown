package engine

import (
	"DownDown/config"
	"os"
	"sync"
)

// Engine 下载引擎
// 全局管理器
type Engine struct {
	Config        config.Config
	DownFileNum   int                      // 下载文件数量
	downFileInfos map[string]*DownFileInfo // 下载文件对象。key为文件名，value为下载文件对象

	downLimit chan struct{} //下载限速器

	resumeIndex *os.File   //断点续传文件索引
	resumeList  []string   //断点续传文件列表
	resumeMu    sync.Mutex //并发保护锁

	DownInfo chan [3]string //下载通道
}

// DownFileInfo 下载文件对象
type DownFileInfo struct {
	FileName string // 文件名
	FileSize int64  // 文件大小

	FilePath string // 文件本地存储路径
	url      string // 文件下载地址

	downManager downManager // 下载管理器

	engine *Engine // 指向引擎

	isResume bool //是否是断点续传文件
}

// downManager 下载管理器
type downManager struct {
	chunkNum  int        // 分片数量
	chunkSize int        // 分片大小
	downChunk int        // 已下载分片数量
	chunks    chan chunk //分片下载队列
	mu        sync.Mutex //并发保护锁

	file       *os.File   //文件对象
	resumeFile *os.File   //恢复文件对象
	reMu       sync.Mutex //并发保护锁

	waitGroup sync.WaitGroup //等待组

	stop   chan struct{} //停止下载信号
	reDown chan chunk    //重新下载信号
}

// chunk 分片
type chunk struct {
	chunkID   int    // 分片ID（从0开始,顺序计算）
	chunkSize int    // 分片大小
	start     int    //分片起始
	end       int    //分片结束
	rangeSize string // 分片在文件中的range

	isDown bool //是否已经下载-仅用于断点续传
}
