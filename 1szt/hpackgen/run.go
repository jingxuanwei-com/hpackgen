package hpackgen

import (
	"1szt/env"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	overridesDir = "data/overrides"
	manifestFile = "data/server-manifest.json"
)

// Manifest 表示 server-manifest.json 结构
type Manifest struct {
	Name        string      `json:"name"`
	Author      string      `json:"author"`
	Version     string      `json:"version"`
	Description string      `json:"description"`
	FileAPI     string      `json:"fileApi"`
	Files       []FileEntry `json:"files"`
	Addons      []Addon     `json:"addons"`
}

// FileEntry 表示文件条目
type FileEntry struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// Addon 表示附加组件（游戏引擎 / 加载器）
type Addon struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// Run 启动 hpackgen 模块
//
// 功能：
//  1. 初始化环境配置（固定配置写入 data/.env）
//  2. 扫描 data/overrides/ 生成初始 server-manifest.json
//  3. 启动文件监听，overrides/ 变动时自动重新生成
func Run() {
	// 1. 初始化固定配置（缺失项自动追加到 data/.env）
	env.Init([][]string{
		{"MANIFEST_NAME", "1szt", "服务器名称"},
		{"MANIFEST_AUTHOR", "1szt", "作者"},
		{"MANIFEST_VERSION", "1szt", "版本号（整合包版本）"},
		{"MANIFEST_DESCRIPTION", "# 欢迎来到 1szt 服务器\\n\\n感谢你选择加入我们的世界。  \\n交流与反馈请前往 QQ 群：565941634\\n", "整合包描述（使用 \\n 换行）"},
		{"MANIFEST_FILE_API", "https://mc.1szt.com", "文件 API 地址"},
		{"MANIFEST_ADDONS", `[{"id":"game","version":"1.21.1"},{"id":"neoforge","version":"21.1.236"}]`, "附加组件（JSON 数组，用于 addons 字段）"},
	})

	fmt.Println("[hpackgen] 环境配置已就绪")

	// 2. 生成初始 manifest
	if err := generateManifest(); err != nil {
		fmt.Printf("[hpackgen] 生成 manifest 失败: %v\n", err)
	} else {
		fmt.Println("[hpackgen] 初始 manifest 已生成")
	}

	// 3. 启动文件监听（后台 goroutine）
	go watchOverrides()
	fmt.Println("[hpackgen] 文件监听已启动，等待 data/overrides/ 变动...")
}

// generateManifest 扫描 overrides/ 目录并生成 server-manifest.json
func generateManifest() error {
	manifest := Manifest{
		Name:        env.GetConfig("MANIFEST_NAME"),
		Author:      env.GetConfig("MANIFEST_AUTHOR"),
		Version:     env.GetConfig("MANIFEST_VERSION"),
		Description: unescapeNewlines(env.GetConfig("MANIFEST_DESCRIPTION")),
		FileAPI:     env.GetConfig("MANIFEST_FILE_API"),
	}

	// 解析 addons JSON
	addonsStr := env.GetConfig("MANIFEST_ADDONS")
	if addonsStr != "" {
		if err := json.Unmarshal([]byte(addonsStr), &manifest.Addons); err != nil {
			fmt.Printf("[hpackgen] 解析 addons 失败: %v\n", err)
		}
	}

	// 扫描 overrides/ 目录，计算文件哈希
	if err := scanOverrides(&manifest); err != nil {
		return fmt.Errorf("扫描 overrides 失败: %w", err)
	}

	// 写入 JSON 文件（美化格式）
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}

	if err := os.WriteFile(manifestFile, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("[hpackgen] 已生成 %s（共 %d 个文件）\n", manifestFile, len(manifest.Files))
	return nil
}

// scanOverrides 遍历 overrides/ 目录，计算每个文件的 SHA1 哈希
func scanOverrides(m *Manifest) error {
	m.Files = nil

	// 检查目录是否存在
	if _, err := os.Stat(overridesDir); os.IsNotExist(err) {
		fmt.Println("[hpackgen] 未找到 overrides/ 目录，跳过文件扫描")
		return nil
	}

	return filepath.Walk(overridesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录本身
		if info.IsDir() {
			return nil
		}

		// 计算相对路径（相对 overrides/）
		relPath, err := filepath.Rel(overridesDir, path)
		if err != nil {
			return err
		}
		// 将反斜杠统一为正斜杠（JSON 中跨平台一致）
		relPath = filepath.ToSlash(relPath)

		// 计算 SHA1 哈希
		hash, err := sha1Hash(path)
		if err != nil {
			fmt.Printf("[hpackgen] 计算哈希失败 %s: %v\n", relPath, err)
			return nil // 跳过失败的文件，不中断整个流程
		}

		m.Files = append(m.Files, FileEntry{
			Path: relPath,
			Hash: hash,
		})
		return nil
	})
}

// sha1Hash 计算文件的 SHA1 哈希值（十六进制字符串）
// HMCL 自动更新使用 SHA1 校验文件完整性
func sha1Hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// watchOverrides 定时检测 overrides/ 变动，自动重新生成 manifest
func watchOverrides() {
	lastSnapshot := takeSnapshot()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		currentSnapshot := takeSnapshot()
		if currentSnapshot != lastSnapshot {
			fmt.Println("[hpackgen] 检测到 data/overrides/ 变动，正在重新生成 manifest...")
			if err := generateManifest(); err != nil {
				fmt.Printf("[hpackgen] 重新生成失败: %v\n", err)
			} else {
				fmt.Println("[hpackgen] manifest 已更新")
			}
			lastSnapshot = currentSnapshot
		}
	}
}

// takeSnapshot 对 overrides/ 目录生成快照字符串，用于检测变动
// 快照包含每个文件的：相对路径:文件大小:最后修改时间(ns)
func takeSnapshot() string {
	if _, err := os.Stat(overridesDir); os.IsNotExist(err) {
		return ""
	}

	var snapshot string
	filepath.Walk(overridesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(overridesDir, path)
		relPath = filepath.ToSlash(relPath)
		snapshot += fmt.Sprintf("%s:%d:%d\n", relPath, info.Size(), info.ModTime().UnixNano())
		return nil
	})

	return snapshot
}

// unescapeNewlines 将字符串中的 \\n 转义序列替换为真正的换行符
func unescapeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	return s
}
