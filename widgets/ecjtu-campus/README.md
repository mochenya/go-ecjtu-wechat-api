# 华交校园（ECJTU Campus）· KDE Plasma 6 小部件

基于本仓库 `ecjtu` CLI 的 KDE Plasma 6 今明课表小部件（`io.github.mochenya.ecjtucampus`），支持**面板弹窗**与**桌面常驻**双形态，采用自研 GLSL 440 液态玻璃材质。

- **作者**：Jensen Chan
- **版本**：`0.1.0`
- **最低要求**：KDE Plasma `6.0`、Qt 6（`Core` / `Gui` / `Qml` / `Quick` / `ShaderTools` / `Effects`）、`KF6WindowSystem`、`cmake`（推荐 `ninja`），以及已安装并在 `PATH` 中可用的 `ecjtu` CLI。

## 核心设计

### 目录与职责

| 路径 | 职责 |
|---|---|
| `package/metadata.json` | Plasmoid 元信息（双语名称/描述、分类、版本与链接） |
| `package/contents/ui/` | QML 视图：`main.qml`（面板弹窗/桌面双入口）、`CampusPanel.qml`、`CourseRow.qml`、`StatusView.qml`、`scheduleUtils.js` 与 `glass/` 光学组件 |
| `shaders/` | 自研 Qt 6 GLSL 440 片元着色器：`liquid_surface.frag`（超椭圆玻璃主壳）、`liquid_pill.frag`（液态分段滑块）、`kawase_blur.frag`（9-tap 高斯柔焦） |
| `plugin/` | C++ QML 模块 `io.github.mochenya.ecjtucampus.backend`：`CourseBackend`（异步调用 CLI 与状态机）、`GlassWindowHelper`（KWin 超椭圆模糊桥接） |
| `scripts/` | 用户级安装（`install.sh`）与卸载（`uninstall.sh`）脚本 |
| `preview/` | 独立离线预览窗口与脚本，使用脱敏假 CLI 调试视觉与状态 |
| `tests/` | 基于 Qt Test 与 `tests/fake-bin/ecjtu` 的离线单元测试 |

### 液态玻璃双渲染管线

- **面板弹窗模式**（`PlasmaCore.Dialog`，`NoBackground`）：
  - `GlassWindowHelper` 通过 `KF6::WindowSystem` 将 `KWindowEffects::enableBlurBehind` 与 `enableBackgroundContrast` 精确裁剪到 56 点 Apple 超椭圆（`n = 3.8`，避开 14px 外投影安全边距），并监听窗口生命周期事件防止 Plasma 主题刷新清掉模糊。
  - `liquid_surface.frag` 仅叠加轻薄中性烟晶滤膜与 **1.5px 135° 光标随动镜面切边**；文字采用纯白字阶配合单 Pass `MultiEffect` 柔和投影，兼顾深色终端、高彩壁纸与纯白网页的可读性。
  - 受 Wayland 跨窗口隔离限制，独立弹窗不直接读取下方其他窗口像素；若系统装有 `kwin-effects-glass` 等兼容标准模糊协议的合成器插件，无需修改本组件即可自动获得跨窗口边缘折射。
- **桌面常驻模式**（采样 `Plasmoid.containment.wallpaperGraphicsObject`）：
  - 直接获取实时壁纸纹理，经 3-Pass 9-tap 高斯奶油柔焦 + 边缘弯曲折射与 RGB 三通道色散 + `1.55×` 活色增强渲染，中心阅读区零畸变。

### 数据、交互与安全边界

- **凭证隔离**：仅从系统环境变量读取 `ECJTU_WEIXIN_TOKEN`，不读取 `.env`，不提供界面输入框，也不向 QML 传递或打印原始 OpenID；错误诊断中自动脱敏为「前 4 位 + `***`」。
- **单进程直调**：`CourseBackend` 以 `Asia/Shanghai` 计算今天/明天，经 `QProcess` 直接执行 `ecjtu course --date YYYY-MM-DD`（不经 shell）；同日加载中自动防重，快速切日丢弃过期返回。
- **忠实呈现**：上游不返回上下课钟点，页面以节次（`periods`）为主轴并按 `1–4 / 5–8 / 9+` 归组为上午/下午/晚间；隐藏常态「上课」角标，仅点亮「实验 / 考试」；点击课程行原地展开完整地点与教学周次。

## 常用脚本与命令

所有命令均在仓库根目录执行：

| 场景 | 命令 | 说明 |
|---|---|---|
| 用户级安装 | `./widgets/ecjtu-campus/scripts/install.sh` | Release 构建并安装到 `~/.local`，首次生成会话环境脚本 |
| 安装并立即生效 | `./widgets/ecjtu-campus/scripts/install.sh --reload` | 安装后同步环境变量到 `systemd --user` 并重启 `plasma-plasmashell.service` |
| 卸载（保留环境脚本） | `./widgets/ecjtu-campus/scripts/uninstall.sh` | 移除 `~/.local` 下的小部件包与 C++ QML 模块 |
| 彻底卸载 | `./widgets/ecjtu-campus/scripts/uninstall.sh --purge-env` | 额外删除 `~/.config/plasma-workspace/env/ecjtu-campus.sh` |
| 构建与离线单测 | `cmake -S widgets/ecjtu-campus -B widgets/ecjtu-campus/build -G Ninja && cmake --build widgets/ecjtu-campus/build && ctest --test-dir widgets/ecjtu-campus/build --output-on-failure` | 编译着色器与 C++ 插件并运行离线 Qt Test |
| 离线视觉预览 | `./widgets/ecjtu-campus/preview/run-preview.sh` | 用脱敏假 `ecjtu` 启动独立预览窗（需 `qml6`，不发真实网络请求） |

### 安装落盘范围（仅当前用户目录，无需 root）

- 小部件包：`~/.local/share/plasma/plasmoids/io.github.mochenya.ecjtucampus/`
- C++ QML 模块：`~/.local/lib/qml/io/github/mochenya/ecjtucampus/backend/`
- 会话环境脚本：`~/.config/plasma-workspace/env/ecjtu-campus.sh`（权限 `600`，二次安装不会覆盖已有内容）

安装脚本**不会**修改 `/usr/`、`/etc/`、`~/.config/kwinrc` 或其他 Plasma 小部件。若桌面会话尚未注入 `ECJTU_WEIXIN_TOKEN`，编辑 `~/.config/plasma-workspace/env/ecjtu-campus.sh` 取消注释并填入 OpenID 后重新登录或运行 `--reload` 即可。

## CLI 退出码与界面状态映射

| 触发条件 | 界面状态 | 处理方式 |
|---|---|---|
| 缺少 `ECJTU_WEIXIN_TOKEN` | `MissingToken`（不启动子进程） | 提示配置环境变量并提供「重新检测」 |
| 未找到 `ecjtu` 可执行文件 | 错误提示（`not_found`） | 提示检查 `ecjtu` 安装与 `PATH` |
| `0` 且 `courses` 非空 / 为空 | 课表列表 / 无课日空态 | 无课日提供一键切换另一天 |
| `2`（参数或环境配置错误） | 配置异常提示 | 展示脱敏诊断与重试入口 |
| `3`（OpenID 未绑定或失效） | 凭证失效提示 | 引导前往微信公众号重新绑定 |
| `4`（网络/超时/上游异常） | 上游暂不可用提示 | 提供手动「重试」，不做自动轮询 |
| `5` 或 JSON 解析失败 | 数据结构异常提示 | 展示脱敏诊断，便于排查上游改版 |
