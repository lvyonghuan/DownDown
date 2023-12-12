package config

type Config struct {
	Proxy     string `mapstructure:"proxy"`
	ChunkSize int    `mapstructure:"chunk_size"` //分块大小
	DownLimit int    `mapstructure:"down_limit"` //下载限速（KB/s）
}
