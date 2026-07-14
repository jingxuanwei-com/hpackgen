package main

import (
	"1szt/hpackgen"
	"1szt/motd"
)

func main() {
	// 欢迎语
	motd.Run()

	// 启动 hpackgen 模块（环境配置 + manifest 生成 + 文件监听）
	hpackgen.Run()

	select {}
}
