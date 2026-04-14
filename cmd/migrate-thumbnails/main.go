package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/sqliteutil"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	dryRun := flag.Bool("dry-run", false, "Preview changes without writing to database")
	force := flag.Bool("force", false, "Run even if thumbnail URLs are already empty")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	db, err := sqliteutil.Open(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// ─── Step 0: Statistics ───
	var totalImages int64
	if err := db.QueryRow("SELECT COUNT(*) FROM images").Scan(&totalImages); err != nil {
		fmt.Fprintf(os.Stderr, "统计图片数量失败: %v\n", err)
		os.Exit(1)
	}

	var withThumbnail int64
	if err := db.QueryRow("SELECT COUNT(*) FROM images WHERE thumbnail_small_url IS NOT NULL AND thumbnail_small_url != ''").Scan(&withThumbnail); err != nil {
		fmt.Fprintf(os.Stderr, "统计缩略图数量失败: %v\n", err)
		os.Exit(1)
	}

	var pendingThumbJobs int64
	if err := db.QueryRow("SELECT COUNT(*) FROM async_jobs WHERE type = 'thumbnail_generate' AND status IN ('ready', 'running')").Scan(&pendingThumbJobs); err != nil {
		fmt.Fprintf(os.Stderr, "统计待处理任务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 缩略图迁移脚本 ===")
	fmt.Printf("数据库: %s\n\n", cfg.Database.Path)
	fmt.Printf("  图片总数:            %d\n", totalImages)
	fmt.Printf("  已有缩略图URL的图片: %d\n", withThumbnail)
	fmt.Printf("  待处理的缩略图任务:  %d\n", pendingThumbJobs)

	if withThumbnail == 0 && totalImages > 0 && !*force {
		fmt.Println("\n所有图片的缩略图URL已为空，无需迁移。使用 --force 强制创建任务。")
		os.Exit(0)
	}

	if totalImages == 0 {
		fmt.Println("\n数据库中没有图片，无需迁移。")
		os.Exit(0)
	}

	fmt.Println()

	// ─── Step 1: Clear old thumbnail URLs ───
	if withThumbnail > 0 {
		fmt.Printf("[Step 1] 清理旧的缩略图URL...")
		if *dryRun {
			fmt.Printf(" (dry-run, 跳过)\n")
			fmt.Printf("  将清理 %d 张图片的 thumbnail_small_url 和 thumbnail_large_url\n", withThumbnail)
		} else {
			result, err := db.Exec(`
				UPDATE images
				SET thumbnail_small_url = NULL,
				    thumbnail_large_url = NULL,
				    updated_at = CURRENT_TIMESTAMP
				WHERE thumbnail_small_url IS NOT NULL OR thumbnail_large_url IS NOT NULL
			`)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n清理缩略图URL失败: %v\n", err)
				os.Exit(1)
			}
			cleared, _ := result.RowsAffected()
			fmt.Printf(" 完成 (已清理 %d 张)\n", cleared)
		}
	} else {
		fmt.Println("[Step 1] 缩略图URL已为空，跳过清理")
	}

	// ─── Step 2: Clean up stale thumbnail_generate jobs ───
	if pendingThumbJobs > 0 {
		fmt.Printf("[Step 2] 清理旧的待处理缩略图任务...")
		if *dryRun {
			fmt.Printf(" (dry-run, 跳过)\n")
			fmt.Printf("  将删除 %d 个 ready/running 状态的 thumbnail_generate 任务\n", pendingThumbJobs)
		} else {
			delResult, err := db.Exec("DELETE FROM async_jobs WHERE type = 'thumbnail_generate' AND status IN ('ready', 'running')")
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n清理旧任务失败: %v\n", err)
				os.Exit(1)
			}
			deleted, _ := delResult.RowsAffected()
			fmt.Printf(" 完成 (已删除 %d 个)\n", deleted)
		}
	} else {
		fmt.Println("[Step 2] 没有旧的待处理缩略图任务，跳过")
	}

	// ─── Step 3: Create thumbnail_generate jobs for all images ───
	fmt.Printf("[Step 3] 为所有图片创建缩略图生成任务...\n")

	rows, err := db.Query("SELECT id, path, filename FROM images ORDER BY id")
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询图片失败: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	created := 0
	for rows.Next() {
		var id int64
		var imgPath, filename string
		if err := rows.Scan(&id, &imgPath, &filename); err != nil {
			fmt.Fprintf(os.Stderr, "读取图片数据失败: %v\n", err)
			os.Exit(1)
		}

		// Filename for thumbnail naming: base name without extension
		// Must match scanner_service.go logic:
		//   filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		thumbFilename := strings.TrimSuffix(filepath.Base(imgPath), filepath.Ext(imgPath))

		payload, err := json.Marshal(map[string]any{
			"image_id": id,
			"path":     imgPath,
			"filename": thumbFilename,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "构造payload失败 (image_id=%d): %v\n", id, err)
			os.Exit(1)
		}

		if *dryRun {
			created++
			continue
		}

		_, err = db.Exec(`
			INSERT INTO async_jobs (platform_task_id, type, status, payload, progress, created_at)
			VALUES (NULL, 'thumbnail_generate', 'ready', ?, 0, CURRENT_TIMESTAMP)
		`, string(payload))
		if err != nil {
			fmt.Fprintf(os.Stderr, "插入任务失败 (image_id=%d): %v\n", id, err)
			os.Exit(1)
		}
		created++
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "遍历图片失败: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Printf("  (dry-run) 将创建 %d 个缩略图生成任务\n", created)
	} else {
		fmt.Printf("  已创建 %d 个缩略图生成任务\n", created)
	}

	// ─── Summary ───
	fmt.Println()
	fmt.Println("=== 迁移完成 ===")
	if *dryRun {
		fmt.Println("（dry-run 模式，未实际修改数据库）")
	}
	fmt.Println()
	fmt.Println("下一步操作:")
	fmt.Println("  1. 确认 config.yaml 中已配置 MinIO:")
	fmt.Println("       thumbnail_storage_provider: \"minio\"")
	fmt.Println("       minio:")
	fmt.Println("         endpoint: \"your-minio-server:9000\"")
	fmt.Println("         access_key: \"...\"")
	fmt.Println("         secret_key: \"...\"")
	fmt.Println("         bucket: \"acg\"")
	fmt.Println("         use_ssl: false")
	fmt.Println("  2. 启动服务，worker 池将自动处理缩略图生成和 MinIO 上传")
	fmt.Println("  3. 可通过管理后台监控任务进度")
}
