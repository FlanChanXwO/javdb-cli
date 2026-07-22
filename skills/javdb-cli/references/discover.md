# 检索与导航工作流

先判断用户是在找影片、人物还是合集，并按需求选择命令：

1. 番号/关键词：`javdb search QUERY --json`。结果可能是 movie、actor、series、maker、director、code 或 list；先读取实际类型和 ID。
2. 单部影片：`javdb detail NUMBER --json`。只有已知内部 ID 时才加 `--id`。详情中的系列、演员、厂牌、导演和标签 ID 可作为下一次实体命令的输入。
3. 实体片单：`javdb actor|series|maker|director|code|list REF --json`。用户要求可下载内容时可加 `--main m --has-magnets`；不要把这两个过滤器误称为全量结果。
4. 主题浏览：先 `javdb tags --zone ZONE` 取得准确标签，再 `javdb browse --tag TAG --json`。需要重建缓存时，用户应明确要求 `tags --refresh`。
5. 合集：用 `javdb lists search QUERY` 找公开合集，用 `javdb list LIST_ID` 读取其中影片。不要将认证的“我的合集”默认命令 `javdb lists` 误作公开搜索。
6. 磁力：账号可用后运行 `javdb magnets NUMBER --json`。用户要求一条推荐结果时才加 `--best`；需要指定条件可用 `--cnsub`、`--hd`、`--min-size`，并在回应中说明过滤条件。

每个阶段先检查退出码；API 返回错误、空结果或认证失败均应如实呈现，而不是更换主机、代理、账号或关键字来“补救”。
