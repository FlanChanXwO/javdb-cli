# Original user input (verbatim)

## Context conversation summary leading to this goal

User had Python javdb-cli complete; requested Go rewrite referencing pixiv-cli, no MCP, ssh-style login, command parity. Plan approved; P0 skeleton landed in sibling path javdb-cli-go.

Then user said (plan mode request):

```
我想我们还缺乏docs，readme.md, contributing.md，changelog.md 以及license。以及我需要你发布到我的仓库，并用我的brew自定义源做下发包。然后我还需要一个github仓库。你可以使用 /github 技能帮你创建仓库发包，但是你需要跑全量测试，以及制作可靠的workflow发包流程，和workflow基础检验。docs绝对不能包含逆向相关资料，只能包含项目的一些基本细节。readme.md必须有免责声明。这些md文档都要有zh,en中英对应的版本
```

When asked first-release scope, user answered:

```
先补全部版本，你先发起一个goal，你基于tdd持续完成，每完成一个模块就审查+测试，然后进行整个系统的全量测试，以确保整个任务进行
```

And repo visibility:

```
Public（推荐）
```

Approved plan path: /Users/flanchan/.claude/plans/reactive-orbiting-emerson.md

## Core requirements (must satisfy)

1. Complete full CLI command parity with Python javdb-cli (not P0-only) before public v0.1.0.
2. Use goal mode + TDD: one module at a time, review + tests each step, full system tests periodically.
3. Bilingual docs EN+ZH: README, CONTRIBUTING, CHANGELOG, docs/* for usage/dev basics.
4. LICENSE (MIT).
5. README must have disclaimer; docs must NOT contain reverse-engineering materials (no APK/Frida/signature RE).
6. Create public GitHub repo FlanChanXwO/javdb-cli.
7. Full test suite + reliable CI quality workflow + release workflow.
8. Publish to Homebrew custom tap FlanChanXwO/homebrew-tap (formula javdb-cli → binary javdb).

## Working tree

/Users/flanchan/Development/SourceCode/GithubProjects/javdb-cli-go

Module: github.com/FlanChanXwO/javdb-cli
Binary: javdb
P0 already done: auth multi-account, config, signature, tls-client appapi client.
