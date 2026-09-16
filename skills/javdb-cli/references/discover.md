# 检索与导航工作流

先判断用户是在找影片、人物还是合集，并按需求选择命令：

1. 番号/关键词：`javdb search QUERY --json`。结果可能是 movie、actor、series、maker、director、code 或 list；先读取实际类型和 ID。需要逐条交给下游命令时使用 `--ndjson`。
2. 单部影片：`javdb detail NUMBER --json`。只有已知内部 ID 时才加 `--id`。详情中的系列、演员、厂牌、导演和标签 ID 可作为下一次实体命令的输入。
3. 实体片单：`javdb actor|series|maker|director|code|list REF --json`。用户要求可下载内容时可加 `--main m --has-magnets`；不要把这两个过滤器误称为全量结果。
4. 主题浏览：先 `javdb tags --zone ZONE` 取得准确标签，再 `javdb browse --tag TAG --json`。需要重建缓存时，用户应明确要求 `tags --refresh`。
5. 合集：用 `javdb lists search QUERY --zone all --ndjson` 找公开合集时，每个结果都是带稳定 `id`/`ref` 且原始对象位于 `data.list` 的 list 信封；缺少稳定 ID 时显式失败，显示名称缺失时 `ref` 回退到 ID。用 `javdb list LIST_ID` 读取其中影片。`lists related` 收到带 movie `id` 的信封时直接使用该 ID。不要将认证的“我的合集”默认命令 `javdb lists` 误作公开搜索。
6. 收藏实体：`javdb collections actors|series|codes|makers|directors --ndjson` 每个实体输出一个对应单数 kind 的信封，原始实体位于 `data.entity`；实体缺少稳定 ID 时显式失败，显示名称缺失时 `ref` 回退到 ID；需要兼容聚合结果时使用 `--json`。
7. 磁力：`javdb magnets NUMBER --json` 无需登录即可获取磁力链接；已保存默认账号 token 时自动带上，token 失效则回退匿名。用户要求一条推荐结果时才加 `--best`；需要指定条件可用 `--cnsub`、`--hd`、`--min-size`，其中 `--min-size` 必须为非负数（零合法，负小数也拒绝），并在回应中说明过滤条件。
8. 排行：`javdb rankings movies|actors|playback --json` 无需登录；`javdb top250 --json` 需要登录。影片与播放排行的结果字段为 `movies`，演员排行为 `actors`。需要仅保留有磁力的影片时加 `--has-magnets`，不要把过滤后结果误称为完整榜单。
9. 评论：`javdb comments NUMBER --page 1 --limit 20 --json` 每次只取所选一页；不要自动读取后续页，也不要把页面评论误称为全量评论。
10. 本地资源：只有用户明确要求写入本机文件时，才执行 `javdb assets list NUMBER` 查看资产（顺序固定：thumbnail、cover、preview 图、preview 视频；编号只是过滤后列表的位置），再用 selector（如 `1-4`、`1,3-5`）或 `--type image|video` 过滤，管道到 `javdb assets download`（`-d DIR` 自动命名，`-o PATH` 仅限单个资产）。该命令域只保存 thumbnail/preview 资源，不下载完整影片或磁力目标；绝不替换已有文件。

每个阶段先检查退出码；API 返回错误、空结果或认证失败均应如实呈现，而不是更换主机、代理、账号或关键字来“补救”。
