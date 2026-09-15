# 未发布

> 此处是可选的人工草稿区，发布 workflow 不会读取本文件；创建 release-prep PR 前，
> 请将最终双语说明放入目标 `changelog/vX.Y.Z/` 目录。

- Machine contract 迁移：lists/collections 的 NDJSON 从聚合 payload 改为逐记录 envelope。
  旧字段为 `data.lists`、`data.items`；新记录分别使用 `data.list`、`data.entity`。
  legacy 人类输出和显式聚合 `--json` 保持不变，list/entity 现在强制要求稳定 ID。
