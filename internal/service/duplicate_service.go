package service

import (
	"context"
	"sort"
	"sync"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// DuplicateService 重复检测服务
type DuplicateService struct {
	imageRepo     repository.ImageRepository
	duplicateRepo repository.DuplicateRepository
	hashService   *HashService
}

// DetectOptions 检测选项
type DetectOptions struct {
	Threshold int // 汉明距离阈值，默认 10
}

// NewDuplicateService 创建重复检测服务实例
func NewDuplicateService(
	imageRepo repository.ImageRepository,
	duplicateRepo repository.DuplicateRepository,
	hashService *HashService,
) *DuplicateService {
	return &DuplicateService{
		imageRepo:     imageRepo,
		duplicateRepo: duplicateRepo,
		hashService:   hashService,
	}
}

// DetectDuplicates 执行重复检测
// 返回检测到的重复组数量
func (s *DuplicateService) DetectDuplicates(ctx context.Context, opts DetectOptions) (int, error) {
	// 设置默认阈值
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 10
	}

	// 清除旧的检测结果
	if err := s.duplicateRepo.DeleteAllDuplicateGroups(); err != nil {
		return 0, err
	}

	// 获取所有图片
	images, err := s.imageRepo.FindAll(1000000, 0, "id", "asc")
	if err != nil {
		return 0, err
	}

	if len(images) == 0 {
		return 0, nil
	}

	// 为每张图片计算哈希
	imageHashes := s.computeHashes(images)

	// 按文件哈希分组（完全相同的图片）
	fileHashGroups := s.groupByFileHash(imageHashes)

	// 使用 Union-Find 进行传递性分组（相似图片）
	similarGroups := s.findSimilarImages(imageHashes, threshold)

	// 合并分组并保存到数据库
	groupCount := s.saveGroups(imageHashes, fileHashGroups, similarGroups, threshold)

	return groupCount, nil
}

// imageHash 图片哈希信息
type imageHash struct {
	image    domain.Image
	fileHash string
	pHash    int64
}

// computeHashes 为图片计算哈希
func (s *DuplicateService) computeHashes(images []domain.Image) []imageHash {
	hashes := make([]imageHash, len(images))
	var wg sync.WaitGroup

	for i, img := range images {
		wg.Add(1)
		go func(idx int, image domain.Image) {
			defer wg.Done()

			hash := imageHash{image: image}

			// 计算文件哈希
			fileHash, err := s.hashService.CalculateFileHash(image.Path)
			if err == nil {
				hash.fileHash = fileHash
			}

			// 计算 pHash（如果尚未计算）
			if image.PHash != 0 {
				hash.pHash = image.PHash
			} else {
				pHash, err := s.hashService.CalculatePHash(image.Path)
				if err == nil {
					hash.pHash = pHash
				}
			}

			hashes[idx] = hash
		}(i, img)
	}

	wg.Wait()
	return hashes
}

// groupByFileHash 按文件哈希分组
func (s *DuplicateService) groupByFileHash(hashes []imageHash) map[string][]imageHash {
	groups := make(map[string][]imageHash)
	for _, h := range hashes {
		if h.fileHash != "" {
			groups[h.fileHash] = append(groups[h.fileHash], h)
		}
	}

	// 只保留有多个图片的组
	result := make(map[string][]imageHash)
	for hash, imgs := range groups {
		if len(imgs) > 1 {
			result[hash] = imgs
		}
	}
	return result
}

// UnionFind 并查集结构
type UnionFind struct {
	parent []int
	rank   []int
}

// NewUnionFind 创建并查集
func NewUnionFind(n int) *UnionFind {
	uf := &UnionFind{
		parent: make([]int, n),
		rank:   make([]int, n),
	}
	for i := 0; i < n; i++ {
		uf.parent[i] = i
		uf.rank[i] = 0
	}
	return uf
}

// Find 查找根节点（带路径压缩）
func (uf *UnionFind) Find(x int) int {
	if uf.parent[x] != x {
		uf.parent[x] = uf.Find(uf.parent[x])
	}
	return uf.parent[x]
}

// Union 合并两个集合
func (uf *UnionFind) Union(x, y int) {
	px, py := uf.Find(x), uf.Find(y)
	if px == py {
		return
	}

	// 按秩合并
	if uf.rank[px] < uf.rank[py] {
		px, py = py, px
	}
	uf.parent[py] = px
	if uf.rank[px] == uf.rank[py] {
		uf.rank[px]++
	}
}

// findSimilarImages 使用 Union-Find 找到相似图片组
func (s *DuplicateService) findSimilarImages(hashes []imageHash, threshold int) [][]int {
	n := len(hashes)
	uf := NewUnionFind(n)

	// 对每对图片比较 pHash
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if hashes[i].pHash == 0 || hashes[j].pHash == 0 {
				continue
			}

			distance := s.hashService.HammingDistance(hashes[i].pHash, hashes[j].pHash)
			if distance <= threshold {
				uf.Union(i, j)
			}
		}
	}

	// 收集分组
	groupMap := make(map[int][]int)
	for i := 0; i < n; i++ {
		root := uf.Find(i)
		groupMap[root] = append(groupMap[root], i)
	}

	// 只返回有多个图片的组
	result := make([][]int, 0)
	for _, indices := range groupMap {
		if len(indices) > 1 {
			result = append(result, indices)
		}
	}

	return result
}

// saveGroups 保存分组到数据库
func (s *DuplicateService) saveGroups(hashes []imageHash, fileHashGroups map[string][]imageHash, similarGroups [][]int, threshold int) int {
	savedGroups := make(map[int64]bool) // 避免重复保存
	count := 0

	// 保存文件哈希组
	for _, imgs := range fileHashGroups {
		if len(imgs) < 2 {
			continue
		}

		// 选择分辨率最高的作为推荐
		recommended := s.selectRecommended(imgs)

		group := &domain.DuplicateGroup{
			RecommendedImageID:  recommended.image.ID,
			SimilarityThreshold: 0, // 文件哈希匹配，阈值为 0
		}

		relations := make([]domain.DuplicateRelation, len(imgs))
		for i, img := range imgs {
			relations[i] = domain.DuplicateRelation{
				ImageID:       img.image.ID,
				IsRecommended: img.image.ID == recommended.image.ID,
				FileHash:      img.fileHash,
				PHashDistance: 0,
			}
		}

		if err := s.duplicateRepo.SaveDuplicateGroup(group, relations); err == nil {
			count++
			for _, img := range imgs {
				savedGroups[img.image.ID] = true
			}
		}
	}

	// 保存相似图片组
	for _, indices := range similarGroups {
		imgs := make([]imageHash, len(indices))
		for i, idx := range indices {
			imgs[i] = hashes[idx]
		}

		if len(imgs) < 2 {
			continue
		}

		// 检查是否已经保存过
		hasNew := false
		for _, img := range imgs {
			if !savedGroups[img.image.ID] {
				hasNew = true
				break
			}
		}
		if !hasNew {
			continue
		}

		// 选择分辨率最高的作为推荐
		recommended := s.selectRecommended(imgs)

		group := &domain.DuplicateGroup{
			RecommendedImageID:  recommended.image.ID,
			SimilarityThreshold: threshold,
		}

		relations := make([]domain.DuplicateRelation, len(imgs))
		for i, img := range imgs {
			distance := s.hashService.HammingDistance(img.pHash, recommended.pHash)
			relations[i] = domain.DuplicateRelation{
				ImageID:       img.image.ID,
				IsRecommended: img.image.ID == recommended.image.ID,
				FileHash:      img.fileHash,
				PHashDistance: distance,
			}
		}

		if err := s.duplicateRepo.SaveDuplicateGroup(group, relations); err == nil {
			count++
		}
	}

	return count
}

// selectRecommended 选择分辨率最高的图片作为推荐
func (s *DuplicateService) selectRecommended(imgs []imageHash) imageHash {
	if len(imgs) == 0 {
		return imageHash{}
	}

	recommended := imgs[0]
	maxResolution := recommended.image.Width * recommended.image.Height

	for _, img := range imgs[1:] {
		resolution := img.image.Width * img.image.Height
		if resolution > maxResolution {
			maxResolution = resolution
			recommended = img
		}
	}

	return recommended
}

// GetDuplicateGroups 获取重复组列表
func (s *DuplicateService) GetDuplicateGroups(limit, offset int) ([]domain.DuplicateGroupWithImages, int64, error) {
	groups, err := s.duplicateRepo.FindDuplicateGroups(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.duplicateRepo.CountDuplicateGroups()
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.DuplicateGroupWithImages, len(groups))
	for i, group := range groups {
		_, relations, err := s.duplicateRepo.FindDuplicateGroupByID(group.ID)
		if err != nil {
			continue
		}

		images := make([]domain.DuplicateImage, len(relations))
		for j, rel := range relations {
			img, err := s.imageRepo.FindByID(rel.ImageID)
			if err != nil {
				continue
			}
			images[j] = domain.DuplicateImage{
				ID:            img.ID,
				Path:          img.Path,
				Filename:      img.Filename,
				Width:         img.Width,
				Height:        img.Height,
				FileSize:      img.FileSize,
				IsRecommended: rel.IsRecommended,
				FileHash:      rel.FileHash,
				PHashDistance: rel.PHashDistance,
			}
		}

		// 按推荐排序，然后按汉明距离排序
		sort.Slice(images, func(i, j int) bool {
			if images[i].IsRecommended != images[j].IsRecommended {
				return images[i].IsRecommended
			}
			return images[i].PHashDistance < images[j].PHashDistance
		})

		result[i] = domain.DuplicateGroupWithImages{
			Group:  group,
			Images: images,
		}
	}

	return result, total, nil
}

// GetDuplicateGroup 获取单个重复组详情
func (s *DuplicateService) GetDuplicateGroup(id int64) (*domain.DuplicateGroupWithImages, error) {
	group, relations, err := s.duplicateRepo.FindDuplicateGroupByID(id)
	if err != nil {
		return nil, err
	}

	images := make([]domain.DuplicateImage, len(relations))
	for i, rel := range relations {
		img, err := s.imageRepo.FindByID(rel.ImageID)
		if err != nil {
			continue
		}
		images[i] = domain.DuplicateImage{
			ID:            img.ID,
			Path:          img.Path,
			Filename:      img.Filename,
			Width:         img.Width,
			Height:        img.Height,
			FileSize:      img.FileSize,
			IsRecommended: rel.IsRecommended,
			FileHash:      rel.FileHash,
			PHashDistance: rel.PHashDistance,
		}
	}

	// 按推荐排序，然后按汉明距离排序
	sort.Slice(images, func(i, j int) bool {
		if images[i].IsRecommended != images[j].IsRecommended {
			return images[i].IsRecommended
		}
		return images[i].PHashDistance < images[j].PHashDistance
	})

	return &domain.DuplicateGroupWithImages{
		Group:  *group,
		Images: images,
	}, nil
}

// DeleteDuplicateGroup 删除重复组记录
func (s *DuplicateService) DeleteDuplicateGroup(id int64) error {
	return s.duplicateRepo.DeleteDuplicateGroup(id)
}
