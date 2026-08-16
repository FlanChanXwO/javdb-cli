# v0.7.2 — 2026-08-16

## 新增

- 新增 `search --magnets N`、稳定的磁力排序与筛选、以图搜磁力，以及影片和命名实体的逐项管道记录输出。 ([`1054964`](https://github.com/FlanChanXwO/javdb-cli/commit/1054964661f213da0ff392ce680ec6d98cf58028), [`32d8c3c`](https://github.com/FlanChanXwO/javdb-cli/commit/32d8c3c46719c1e900cfb1d22198610cf2d35bdb), [`b8a9ee5`](https://github.com/FlanChanXwO/javdb-cli/commit/b8a9ee5e5655c091095fed4ad49988648e533242), [`ce93242`](https://github.com/FlanChanXwO/javdb-cli/commit/ce932429c98d27cb7dfcb8c345a8fa1a1f0203ca), [`d7c3f93`](https://github.com/FlanChanXwO/javdb-cli/commit/d7c3f93c0400843637cf3383eccfb8e347305db5), [`cd723be`](https://github.com/FlanChanXwO/javdb-cli/commit/cd723be0a8e7f0da4832dea5222aa4ca5f34a2ea))

## 修复

- 修复非 TTY producer 输出不可稳定消费、上游错误信封被重复处理、重复影片详情请求，以及部分磁力和输出失败未正确报告的问题。 ([`0fb52c6`](https://github.com/FlanChanXwO/javdb-cli/commit/0fb52c613d412ac71126419dcbf82fcbea450b14), [`23ec4e4`](https://github.com/FlanChanXwO/javdb-cli/commit/23ec4e4e39cc1ae46d238f3949225612b5918909), [`2ab4a75`](https://github.com/FlanChanXwO/javdb-cli/commit/2ab4a7568344f0e1216b71499976a8b9c577c6a5), [`2f9b0e9`](https://github.com/FlanChanXwO/javdb-cli/commit/2f9b0e91aa8169b4c3881bfd85aacfab23a159f5), [`472e787`](https://github.com/FlanChanXwO/javdb-cli/commit/472e787d49db7925be0987b26a8f967bd76cae85), [`79ab9dd`](https://github.com/FlanChanXwO/javdb-cli/commit/79ab9dd385872d2ba665ec76ac2f79dd7bac45cb), [`af0e834`](https://github.com/FlanChanXwO/javdb-cli/commit/af0e83428c4db92a82bc6d2bd055d294934f2e34), [`aff20fb`](https://github.com/FlanChanXwO/javdb-cli/commit/aff20fb6863d2ef71d02a8672bf3d57dbbdf2f53), [`f6cae5b`](https://github.com/FlanChanXwO/javdb-cli/commit/f6cae5be5a9d9d42553a166442d92f5e5c641060))

## 文档

- 补充 TTY 感知的输出契约，并为默认文本管道和 producer 投影增加聚焦验收覆盖。 ([`53d4294`](https://github.com/FlanChanXwO/javdb-cli/commit/53d42949f3f39177b22ac1d9c802ce5651ec7b27), [`728f369`](https://github.com/FlanChanXwO/javdb-cli/commit/728f3695621ecd310e0dc43a815cf171c8b9b1bb), [`180e55a`](https://github.com/FlanChanXwO/javdb-cli/commit/180e55a645969a473de5e794099c57e67c63dfee), [`cfa62f6`](https://github.com/FlanChanXwO/javdb-cli/commit/cfa62f6d86fa81fe269be0bb257d41a4cc00cbf4), [`98ecb82`](https://github.com/FlanChanXwO/javdb-cli/commit/98ecb82c5fa727067c8b9b3abb1bec72dc78bce3))

**完整变更**：[v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)
