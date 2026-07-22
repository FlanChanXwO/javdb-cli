# 安装与版本检查

只在用户明确要求安装或修复 `javdb` 时使用本页。先确认平台与目标版本，安装会写入
用户系统路径或包管理器状态，不能由普通查询隐式触发。

优先级：

1. 已发布版本：下载与平台匹配的 GitHub Release 归档，校验 `checksums.txt` 后将 `javdb`
   （Windows 为 `javdb.exe`）放到用户指定的 PATH 目录。
2. macOS/Linux：用户明确选用 Homebrew 时，执行 `brew install FlanChanXwO/tap/javdb-cli`。
3. 源码构建：在仓库根目录执行 `sh scripts/build.sh`，产物为 `build/javdb`。

安装后运行 `javdb version --json`，只报告版本、提交和构建时间，不读取账号文件。
