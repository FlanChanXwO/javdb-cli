# Dockerfile — javdb-cli 容器镜像
#
# 从与原生生产构建相同的不可变 release tag 构建版本化 Linux 二进制后，
# 拷贝到 pinned Debian slim runtime。容器使用同一 javdb 二进制和 ~/.javdb-cli
# 状态命名空间，不创建 Docker 专有产品行为。
#
# 构建上下文预期包含 dist/javdb（release workflow 从不可变 tag 重建）。
#
# 使用方式：
#   docker build -t javdb-cli .
#   docker run --rm javdb-cli --version
#   docker run --rm -v javdb-cli-state:/home/javdb/.javdb-cli javdb-cli login

# Debian bookworm-slim（glibc），pinned 不可变 multi-arch manifest digest，
# 保证构建可复现；不使用可变 tag。
FROM debian@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171 AS runtime

# 安装运行时必要材料：ca-certificates 用于 HTTPS 连接 JavDB App API，
# tzdata 用于时区处理；Debian slim 默认不含这些。
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

# 创建专用非 root 用户 javdb（UID 1000），HOME=/home/javdb。
# 容器不以 root 运行，避免提权风险。
RUN useradd --home-dir /home/javdb --create-home --shell /usr/sbin/nologin --uid 1000 javdb

# 预创建状态目录并固定属主；首次挂载空命名 volume 时 Docker 会继承该 ownership。
RUN mkdir -p /home/javdb/.javdb-cli && chown -R javdb:javdb /home/javdb

# 拷贝预构建的版本化 javdb 二进制（从构建上下文的 dist/javdb 拷入）。
COPY dist/javdb /usr/local/bin/javdb
RUN chmod 0755 /usr/local/bin/javdb

# 携带项目许可证，保证容器分发满足保留版权和许可声明的要求。
COPY LICENSE /usr/share/doc/javdb-cli/
RUN chmod -R a+rX /usr/share/doc/javdb-cli

# 设置 HOME 环境变量，使 javdb CLI 本地状态路径解析到用户 home 目录下。
ENV HOME=/home/javdb

# /work 是默认工作目录，用于下载产物等 bind mount。
WORKDIR /work

# OCI provenance 元数据标签。
# CI 必须注入触发 release 的不可变 tag 与 tag commit；本地构建可显式传入对应值。
ARG REVISION
ARG VERSION
LABEL org.opencontainers.image.source="https://github.com/FlanChanXwO/javdb-cli"
LABEL org.opencontainers.image.revision="${REVISION}"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.licenses="MIT"

# 切换到非 root 用户。
USER javdb

# javdb CLI 入口点——不使用 wrapper script，直接执行二进制。
ENTRYPOINT ["/usr/local/bin/javdb"]
