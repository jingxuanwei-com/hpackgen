// env 轻量级 .env 配置文件管理工具
//
// 用法：
//
//	// 1️⃣ 初始化配置（自动创建 .env，缺失项自动追加）
//	//    下标[0]=key, [1]=value, [2+]=多行注释（自动以 # 写入）
//	env.Init([][]string{
//		{"CHI_PORT", "9081", "端口"},
//		{"GRPC_PORT", "50051", "gRPC 端口"},
//		{"DB_HOST", "127.0.0.1", "--- [网络配置] ---", "仅针对 pgsql/mysql，sqlite 请忽略"},
//	})
//
//	// 2️⃣ 读取值（环境变量优先，没有再读配置文件）
//	port := env.Get("CHI_PORT")
//
//	// 3️⃣ 仅从配置文件读取（忽略系统环境变量）
//	port := env.GetConfig("CHI_PORT")
//
//	// 4️⃣ 运行时修改或新增配置
//	env.Set("CHI_PORT", "9090")
//
// 特性：
//   - 零依赖，~180 行
//   - Init() 自动生成带注释的 .env 文件
//   - 每次 Get() 直接读文件，天然热加载
//   - 并发不安全（配置几乎不改动，无影响）
package env

import (
	"bufio"
	"fmt"

	"os"
	"path/filepath"
	"strings"
)

// fileName .env 文件路径（程序当前目录）
const fileName = ".env"

// Init 初始化环境配置系统
//
// 核心逻辑：
// 1. [发现]：检查 .env 是否存在，不存在则自动创建。
// 2. [对比]：读取现有文件，若参数(Key)已存在，则保持原样
// 3. [追加]：若参数(Key)缺失，则按顺序在文件尾部追加【注释+配置】。
//
// 数组输入规范：
//
//	{ "KEY", "VALUE", "描述1", "描述2", "..." }
//	- 下标[0]: 环境变量名 (如 "WEB_PORT")
//	- 下标[1]: 默认数值 (如 "9081")
//	- 下标[2+]: 任意多行注释，将以 # 开头写入文件
func Init(config [][]string) {
	// 确保 .env 所在目录存在
	dir := filepath.Dir(fileName)
	if dir != "." {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				fmt.Println("创建目录失败:", err)
				return
			}
		}
	}

	// 2. 读取现有内容（如果文件不存在，contentStr 为空，不影响后续判断）
	existingContent, _ := os.ReadFile(fileName)
	contentStr := string(existingContent)

	// 3. 打开文件（如果不存在则创建，存在则准备追加）
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("打开/创建文件失败:", err)
		return
	}
	defer f.Close()

	// 4. 遍历配置数组
	for i := 0; i < len(config); i++ {
		line := config[i]

		// 从line2开始遍历提取注释
		comment := ""
		for j := 2; j < len(line); j++ {
			comment += "# " + line[j] + "\n"
		}

		key := line[0] // 参数名
		val := line[1] // 数据内容

		// 检查 line[1] (Key) 是否已存在
		if strings.Contains(contentStr, key+"=") {
			// 如果已有该参数，跳过不处理
			continue
		}

		// 如果没有，则在文件尾部顺序写入
		// 按照你要求的格式：#注释 \n Key=Value \n\n
		data := fmt.Sprintf("%s%s=%s\n\n", comment, key, val)

		_, err := f.WriteString(data)
		if err != nil {
			fmt.Printf("追加参数 [%s] 失败: %v\n", key, err)
		}
	}
}

// Get 读取值：环境变量优先级高于配置文件
//
// 1. [环境变量]：先从系统环境变量中获取
// 2. [配置文件]：如果环境变量中没有，再从配置文件中读取
// 读取到的值为空时，返回空字符串
func Get(key string) string {
	// 1. 【新增】首先尝试从系统环境变量获取
	// 比如你在 Linux 下执行: WEB_PORT=8080 ./main
	if envVal := os.Getenv(key); envVal != "" {
		return envVal
	}

	// 2. 如果环境变量没有，再打开文件读取
	f, err := os.Open(fileName)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// 过滤注释
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// 匹配 Key=
		if strings.HasPrefix(line, key+"=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}

	// 检查扫描过程是否有错误（如文件读取中断）
	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

// GetConfig 仅从配置文件读取值，不检查系统环境变量
// 适用于明确需要读取 .env 文件内容的场景
func GetConfig(key string) string {
	f, err := os.Open(fileName)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, key+"=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

// Set 修改或新增配置项
//
// 逻辑：
//  1. 读取现有文件，逐行扫描
//  2. 找到 key= 则替换为新的 key=value（保留注释）
//  3. 没找到则在文件末尾追加 key=value
//  4. 写回文件
func Set(key, value string) error {
	// 1. 读取文件全部内容
	existingContent, err := os.ReadFile(fileName)
	if err != nil {
		// 文件不存在，直接创建新文件写入
		return os.WriteFile(fileName, []byte(key+"="+value+"\n"), 0644)
	}

	lines := strings.Split(string(existingContent), "\n")
	found := false

	// 2. 逐行扫描，替换已有 key
	for i, line := range lines {
		// 跳过注释和空行
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		// 匹配 key=
		if strings.HasPrefix(line, key+"=") {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}

	// 3. 没找到则追加
	if !found {
		lines = append(lines, key+"="+value)
	}

	// 4. 写回文件
	return os.WriteFile(fileName, []byte(strings.Join(lines, "\n")), 0644)
}
