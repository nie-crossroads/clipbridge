# ClipBridge 项目协作规则

## Git 提交信息

提交标题使用 `<类型>：<简短中文说明>`。只写本次改动实际涉及的类型，不列空项。标题使用最能概括主要改动的类型，确有必要时在提交正文中补充其他实际涉及的类型。

| 类型 | 适用场景 |
| --- | --- |
| `refactor` | 重构代码 |
| `docs` | 新增或优化文档 |
| `chore` | 日常维护，如升级 SDK、移除废弃代码 |
| `style` | 样式调整 |
| `fix` | 修复 Bug |
| `feat` | 新增功能或节点 |
| `perf` | 优化代码性能 |

提交标题示例为 `fix：修复解析结果为空时的显示错误`。

## GitHub 贡献者署名

本项目的 GitHub Contributors 仅保留项目作者的署名。提交使用项目作者认可、且与作者 GitHub 账号关联的 Git 姓名和邮箱（包括该账号的 GitHub noreply 邮箱）。提交前核对实际生效的作者和提交者身份。

禁止将 Cursor Agent、WorkBuddy Agent 或任何其他 AI Agent、机器人账号写入提交的 Author、Committer 或 `Co-authored-by` 字段。提交信息与尾注也不得添加 Agent 署名、生成工具签名或归属声明。若作者身份尚未配置或无法确认，先确认身份再提交，禁止自行使用 Agent 身份或猜测作者邮箱。
