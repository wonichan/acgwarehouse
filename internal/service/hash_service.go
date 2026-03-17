package service

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/corona10/goimagehash"
	_ "golang.org/x/image/webp"
)

// HashService 提供文件哈希和感知哈希计算功能
type HashService struct{}

// NewHashService 创建新的哈希服务实例
func NewHashService() *HashService {
	return &HashService{}
}

// CalculateFileHash 计算文件的 SHA256 哈希值
// 返回 64 个字符的十六进制字符串
func (s *HashService) CalculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CalculatePHash 计算图片的感知哈希值
// 返回 int64 类型的 pHash 值
// 支持 JPG/PNG/WebP/GIF 格式
func (s *HashService) CalculatePHash(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}

	phash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return 0, err
	}

	return int64(phash.GetHash()), nil
}

// HammingDistance 计算两个 int64 哈希值的汉明距离
// 返回两个哈希值之间不同位的数量
func (s *HashService) HammingDistance(hash1, hash2 int64) int {
	xor := uint64(hash1) ^ uint64(hash2)
	return popcount(xor)
}

// popcount 计算一个 uint64 中置位（1）的数量
func popcount(x uint64) int {
	count := 0
	for x != 0 {
		count += int(x & 1)
		x >>= 1
	}
	return count
}
