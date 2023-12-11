package config

type Config struct {
	Proxy     string
	ChunkSize int `json:"chunk_size"` //分块大小
	DownLimit int `json:"down_limit"` //下载限速（KB/s）
}
